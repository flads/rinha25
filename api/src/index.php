<?php

use Predis\Autoloader;
use Predis\Client;

require __DIR__ . '/../vendor/autoload.php';
require __DIR__ . '/Utils/DateTimeUtils.php';
require __DIR__ . '/Utils/Http.php';

class App
{
    const PAYMENT_PROCESSOR_DEFAULT_URL = 'http://payment-processor-default:8080';
    const PAYMENT_PROCESSOR_FALLBACK_URL = 'http://payment-processor-fallback:8080';

    private Client $client;
    private Http $httpClient;

    private ?string $method = null;
    private ?string $path = null;
    private ?array $params = null;

    private array $routes = [
        'GET' => [
            '/payments-summary' => 'paymentsSummary'
        ],
        'POST' => [
            '/payments' => 'payments',
        ],
    ];

    public function __construct()
    {
        Autoloader::register();

        $this->client = new Predis\Client([
            'scheme' => 'tcp',
            'host'   => 'redis',
            'port'   => 6379,
        ]);

        $this->httpClient = new Http();

        $path = $_SERVER['REQUEST_URI'] ?? '/';

        $this->method = $_SERVER['REQUEST_METHOD'];
        $this->path = explode('?', $path)[0];

        if ($this->method === 'GET') {
            $this->params = $_GET;
        }

        $this->setErrorHandler();
        $this->validateRouteExists();
    }

    public function resolve()
    {
        return $this->{$this->routes[$this->method][$this->path]}();
    }

    private function setErrorHandler()
    {
        set_error_handler(function (int $error, string $message, string $filename, int $line) {
            http_response_code(500);

            echo json_encode([
                'error' => [
                    'message' => $message
                ]
            ]);

            die();
        });
    }

    private function validateRouteExists()
    {
        if (
            !array_key_exists($this->method, $this->routes) ||
            !array_key_exists($this->path, $this->routes[$this->method])
        ) {
            $this->httpClient->response([
                'message' => 'Route does not exist!'
            ], 404);
            die();
        }
    }

    private function payments()
    {
        $body = file_get_contents('php://input');

        $data = [...json_decode($body, true), 'requestedAt' => date('c')];

        $this->client->rpush('requests', json_encode($data));

        return $this->httpClient->response();
    }

    private function paymentsSummary()
    {
        $defaultRequests = $this->getRequests('default_requests');
        $fallbackRequests = $this->getRequests('fallback_requests');

        return $this->httpClient->response([
            'default' => [
                "totalRequests" => count($defaultRequests),
                "totalAmount" => $this->getRequestsAmountSum($defaultRequests),
            ],
            'fallback' => [
                "totalRequests" => count($fallbackRequests),
                "totalAmount" => $this->getRequestsAmountSum($fallbackRequests),
            ],
        ]);
    }

    private function getRequests(string $listName): array
    {
        if (!empty($this->params)) {
            $from = DateTimeUtils::strToTimeWithMicro($this->params['from']);
            $to = DateTimeUtils::strToTimeWithMicro($this->params['to']);

            if (isset($from) && isset($to)) {
                return $this->client->zrangebyscore($listName, $from, $to);
            }
        }

        return $this->client->zrange($listName, 0, -1);
    }

    private function getRequestsAmountSum(array $requests): float
    {
        $amountSum = 0;
        
        foreach ($requests as $request) {
            $data = json_decode($request, true);
            
            if (isset($data['amount'])) {
                $amountSum += $data['amount'];
            }
        }

        return round($amountSum, 2);
    }
}

(new App())->resolve();