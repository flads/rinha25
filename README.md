# Rinha de Backend 2025 - PHP e Go

Projeto feito para a [Rinha de Backend 2025](https://github.com/zanfranceschi/rinha-de-backend-2025).

##### Tecnologias utilizadas:

* [PHP](https://www.php.net/releases/8.4/en.php) - Linguagem de programação.
* [Go](https://go.dev/) - Linguagem de programação.
* [Fast HTTP](https://github.com/valyala/fasthttp) - Servidor HTTP de alto desempenho em Go.
* [Redis](https://redis.io/) - Banco de dados em memória.
* [Nginx](https://nginx.org/) - Proxy reverso e balanceador de carga.

##### Solução

A tabela abaixo mostra como os recursos foram alocados:

| Serviço      | Linguagem | CPU  | Memória   |
|--------------|-----------|------|-----------|
| **api-1**    | Go        | 0.15 | 50MB      |
| **api-2**    | Go        | 0.15 | 50MB      |
| **worker-1** | PHP       | 0.3  | 75MB      |
| **worker-2** | PHP       | 0.3  | 75MB      |
| **nginx**    | -         | 0.3  | 50MB      |
| **redis**    | -         | 0.3  | 50MB      |
| **Total**    | -         | 1.5  | 350MB     |

##### Arquitetura dos Serviços

- **HAProxy**: Responsável por receber as requisições HTTP e atuar como **load balancer**, distribuindo as solicitações entre as duas instâncias da API via soquetes Unix.
- **APIs (api-1 e api-2)**: Gerenciam as requisições de pagamento, enviando os dados para uma fila ordenada no Redis.
- **Worker (worker-1 e worker-2)**: Executa continuamente, consumindo as requests da fila do Redis e processando os pagamentos, enviando requisições aos servidores externos de processamento.
- **Redis**: Banco de dados em memória utilizado como meio de comunicação entre as apis e os workers.
