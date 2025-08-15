<?php

require __DIR__ . '/../vendor/autoload.php';
require __DIR__ . '/Utils/DateTimeUtils.php';
require __DIR__ . '/Utils/Http.php';

use Predis\Autoloader;
use Predis\Client;
use Ramsey\Uuid\Uuid;

class Worker
{
    const PAYMENT_PROCESSOR_DEFAULT_URL = 'http://payment-processor-default:8080';
    const PAYMENT_PROCESSOR_FALLBACK_URL = 'http://payment-processor-fallback:8080';

    private bool $hasFailedRequestsToProcess = false;

    private Client $redis;
    private Http $http;

    public function __construct()
    {
        Autoloader::register();
        
        $this->redis = new Client([
            'scheme' => 'tcp',
            'host'   => 'redis',
            'port'   => 6379,
        ]);

        $this->http = new Http();
    }

    public function execute(): void
    {
        usleep(250000);

        while (true) {
            usleep(10000);
            $requests = (array) $this->redis->lpop('requests', 250);

            foreach ($requests as $request) {
                if (!$request) {
                    break;
                }

                $this->callDefaultProcessor($request);
            }

            if (empty($requests) && $this->hasFailedRequestsToProcess) {
                $this->processFailedRequests();
            }
        }
    }

    private function sendToProcessorApi(string $url, array $data): void
    {
        $payload = json_encode([
            'timestamp' => DateTimeUtils::strToTimeWithMicro($data['requestedAt']),
            'amount'    => (float) $data['amount'],
        ]);

        $this->http->post($url, $payload);
    }

    private function callDefaultProcessor(string $request): bool
    {
        $data = json_decode($request, true);

        $response = $this->http->post(
            self::PAYMENT_PROCESSOR_DEFAULT_URL . '/payments',
            $request
        );

        $isResponseOK = $response['statusCode'] === 200;

        if ($isResponseOK) {
            $this->sendToProcessorApi('http://database:8081/processor-default', $data);

            return $isResponseOK;
        }

        $this->redis->rpush('failed_requests', $request);
        $this->redis->set('default_failed_10_secs_ago', true, 'EX', 10);
        $this->hasFailedRequestsToProcess = true;

        return $isResponseOK;
    }

    private function callFallbackProcessor(string $request): bool
    {
        $data = json_decode($request, true);

        $response = $this->http->post(
            self::PAYMENT_PROCESSOR_FALLBACK_URL . '/payments',
            $request
        );

        $isResponseOK = $response['statusCode'] === 200;

        if ($isResponseOK) {
            $this->sendToProcessorApi('http://database:8081/processor-fallback', $data);
        }

        return $isResponseOK;
    }


    private function processFailedRequests(): void
    {
        $defaultFailed10SecsAgo = $this->redis->get('default_failed_10_secs_ago');

        if (!$defaultFailed10SecsAgo) {
            $failedRequests = (array) $this->redis->lpop('failed_requests', 250);

            if (empty($failedRequests)) {
                $this->hasFailedRequestsToProcess = false;
                return;
            }

            foreach ($failedRequests as $failedRequest) {
                $isResponseOK = $this->callDefaultProcessor($failedRequest);

                if ($isResponseOK) {
                    continue;
                }

                $isFallbackResponseOK = $this->callFallbackProcessor($failedRequest);

                if ($isFallbackResponseOK) {
                    continue;
                }

                $this->redis->rpush('failed_requests', $failedRequest);
            }
        }
    }
}

(new Worker())->execute();