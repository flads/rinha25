<?php

require __DIR__ . '/../vendor/autoload.php';
require __DIR__ . '/Utils/DateTimeUtils.php';
require __DIR__ . '/Utils/Http.php';

use Predis\Autoloader;
use Predis\Client;
use Revolt\EventLoop;

class Worker
{
    const EVENT_LOOP_QUANTITY = 12;
    const EVENT_LOOP_SECONDS = 1;
    const PAYMENT_PROCESSOR_DEFAULT_URL = 'http://payment-processor-default:8080';
    const PAYMENT_PROCESSOR_FALLBACK_URL = 'http://payment-processor-fallback:8080';

    private Client $client;
    private Http $httpClient;

    public function __construct()
    {
        Autoloader::register();
        
        $this->client = new Client([
            'scheme' => 'tcp',
            'host'   => 'redis',
            'port'   => 6379,
        ]);

        $this->httpClient = new Http();
    }

    public function execute(): void
    {
        $suspension = EventLoop::getSuspension();

        for ($i=0; $i < self::EVENT_LOOP_QUANTITY; $i++) {
            if ($i > 0) {
                usleep(37500);
            }

            EventLoop::repeat(
                self::EVENT_LOOP_SECONDS,
                function (): void {
                    $this->makeEvent();
                }
            );
        }

        $suspension->suspend();
    }

    public function makeEvent(): void
    {
        for ($i=0; $i < 250; $i++) {
            $request = $this->client->lpop('requests');

            if (!$request) {
                break;
            }

            $this->makePayment(json_decode($request, true));
        }
    }

    public function makePayment(array $data): void
    {
        $response = $this->httpClient->post(
            self::PAYMENT_PROCESSOR_DEFAULT_URL . '/payments',
            $data
        );

        if ($response['statusCode'] === 200) {
            $this->addToRequestsLists('default_requests', $data);
        }

        if ($response["statusCode"] !== 200) {
            $response = $this->httpClient->post(
                self::PAYMENT_PROCESSOR_FALLBACK_URL . '/payments',
                $data
            );

            if ($response['statusCode'] === 200) {
                $this->addToRequestsLists('fallback_requests', $data);
            }
        }
    }

    private function addToRequestsLists(string $listName, array $data): void
    {
        $this->client->zadd($listName, [
            json_encode($data) => DateTimeUtils::strToTimeWithMicro($data['requestedAt'])
        ]);
    }
}

(new Worker())->execute();