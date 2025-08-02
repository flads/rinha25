<?php

require __DIR__ . '/../vendor/autoload.php';
require __DIR__ . '/Utils/DateTimeUtils.php';
require __DIR__ . '/Utils/Http.php';

use Predis\Autoloader;
use Predis\Client;

class Worker
{
    const PAYMENT_PROCESSOR_DEFAULT_URL = 'http://payment-processor-default:8080';
    const PAYMENT_PROCESSOR_FALLBACK_URL = 'http://payment-processor-fallback:8080';

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
            $requests = (array) $this->redis->lpop('requests', 250);

            foreach ($requests as $request) {
                if (!$request) {
                    break;
                }

                $this->makePayment(json_decode($request, true));
            }
        }
    }

    public function makePayment(array $data): void
    {
        $response = $this->http->post(
            self::PAYMENT_PROCESSOR_DEFAULT_URL . '/payments',
            $data
        );

        if ($response['statusCode'] === 200) {
            $this->addToRequestsLists('default_requests', $data);
            return;
        }

        $response = $this->http->post(
            self::PAYMENT_PROCESSOR_FALLBACK_URL . '/payments',
            $data
        );

        if ($response['statusCode'] === 200) {
            $this->addToRequestsLists('fallback_requests', $data);
        }
    }

    private function addToRequestsLists(string $listName, array $data): void
    {
        $this->redis->zadd($listName, [
            json_encode($data) => DateTimeUtils::strToTimeWithMicro($data['requestedAt'])
        ]);
    }
}

(new Worker())->execute();