<?php

class Http
{
    private array $options = [];

    public function __construct()
    {
        $this->options = [
            'http' => [
                'header' => "Content-type: application/json\r\n",
                'timeout' => 60,
                'ignore_errors' => true
            ],
        ];
    }

    public function get(string $url, ?array $data = null): ?array
    {
        return $this->request('GET', $url, $data);
    }

    public function post(string $url, ?string $data = null): ?array
    {
        try {
            return $this->request('POST', $url, $data);
        } catch (\Throwable $th) {
            throw $th;
        }
    }

    private function request(
        string $method,
        string $url,
        ?string $data = null
    ): ?array
    {
        $this->options['http']['method'] = $method;
        
        if ($data) {
            $this->options['http']['content'] = $data;
        }

        $response = file_get_contents(
            $url,
            false,
            stream_context_create($this->options)
        );

        $responseData = json_decode($response, true);

        error_log("response: " . print_r($responseData, true));

        if (
            !$responseData ||
            (isset($responseData['message']) && $responseData['message'] !== 'payment processed successfully')
        ) {
            return [
                'data' => json_decode($response, true),
                'statusCode' => $this->getResponseStatusCode() ?? 500
            ];
        }

        return [
            'statusCode' => $this->getResponseStatusCode() ?? 200
        ];
    }

    public function response(
        ?array $data = null,
        int $responseCode = 200
    ): void
    {
        header('Content-Type: application/json; charset=utf-8');

        if ($data) {
            http_response_code($responseCode);
            echo json_encode($data);
            return;
        }

        http_response_code(204);
        return;
    }

    private function getResponseStatusCode(): ?int
    {
        if (isset($http_response_header)) {
            preg_match('/([0-9])\d+/', $http_response_header[0], $responseHeader);

            if ($responseHeader) {
                return intval($responseHeader[0]);
            }
        }

        return null;
    }
}