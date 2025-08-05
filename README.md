# Rinha de Backend 2025

Projetos feitos para a [Rinha de Backend 2025](https://github.com/zanfranceschi/rinha-de-backend-2025).

- [PHP](https://github.com/flads/rinha25/tree/master?tab=readme-ov-file#php)
- [PHP e Go](https://github.com/flads/rinha25/tree/master?tab=readme-ov-file#php-e-go)
- [Go](https://github.com/flads/rinha25/tree/master?tab=readme-ov-file#go)

### PHP

| branch: [php](https://github.com/flads/rinha25/tree/php)

##### Tecnologias utilizadas:

* [PHP](https://www.php.net/releases/8.4/en.php) - Linguagem de programação.
* [Redis](https://redis.io/) - Banco de dados em memória.
* [Nginx](https://nginx.org/) - Servidor web HTTP e load balancer.

##### Solução

A tabela abaixo mostra como os recursos foram alocados:

| Serviço      | Linguagem | CPU  | Memória   |
|--------------|-----------|------|-----------|
| **api-1**    | PHP       | 0.25 | 50MB      |
| **api-2**    | PHP       | 0.25 | 50MB      |
| **worker-1** | PHP       | 0.30 | 102.5MB   |
| **worker-2** | PHP       | 0.30 | 102.5MB   |
| **nginx**    | -         | 0.15 | 15MB      |
| **redis**    | -         | 0.25 | 30MB      |
| **Total**    | -         | 1.50 | 350MB     |

### PHP e Go

| branch: [php-go](https://github.com/flads/rinha25/tree/php-go)

##### Tecnologias utilizadas:

* [PHP](https://www.php.net/releases/8.4/en.php) - Linguagem de programação.
* [Go](https://go.dev/) - Linguagem de programação.
* [Gin](https://gin-gonic.com/) - The fastest full-featured web framework for Go. Crystal clear.
* [Redis](https://redis.io/) - Banco de dados em memória.
* [HAProxy](https://www.haproxy.org/) - Proxy reverso e balanceador de carga de alto desempenho.

##### Solução

A tabela abaixo mostra como os recursos foram alocados:

| Serviço      | Linguagem | CPU  | Memória   |
|--------------|-----------|------|-----------|
| **api-1**    | Go        | 0.15 | 45MB      |
| **api-2**    | Go        | 0.15 | 45MB      |
| **worker-1** | PHP       | 0.35 | 90MB      |
| **worker-2** | PHP       | 0.35 | 90MB      |
| **haproxy**  | -         | 0.30 | 50MB      |
| **redis**    | -         | 0.20 | 30MB      |
| **Total**    | -         | 1.50 | 350MB     |

### Go

| branch: [go](https://github.com/flads/rinha25/tree/go)

##### Tecnologias utilizadas:

* [Go](https://go.dev/) - Linguagem de programação.
* [Gin](https://gin-gonic.com/) - The fastest full-featured web framework for Go. Crystal clear.
* [Redis](https://redis.io/) - Banco de dados em memória.
* [HAProxy](https://www.haproxy.org/) - Proxy reverso e balanceador de carga de alto desempenho.

##### Solução

A tabela abaixo mostra como os recursos foram alocados:

| Serviço      | Linguagem | CPU  | Memória   |
|--------------|-----------|------|-----------|
| **api-1**    | Go        | 0.32 | 45MB      |
| **api-2**    | Go        | 0.32 | 45MB      |
| **worker-1** | Go        | 0.18 | 90MB      |
| **worker-2** | Go        | 0.18 | 90MB      |
| **haproxy**  | -         | 0.30 | 50MB      |
| **redis**    | -         | 0.20 | 30MB      |
| **Total**    | -         | 1.50 | 350MB     |

#### Arquitetura dos Serviços

- **HAProxy**: Responsável por receber as requisições HTTP e atuar como **load balancer**, distribuindo as solicitações entre as duas instâncias da API via soquetes Unix.
- **APIs (api-1 e api-2)**: Gerenciam as requisições de pagamento, enviando os dados para uma fila ordenada no Redis.
- **Worker (worker-1 e worker-2)**: Executa continuamente, consumindo as requests da fila do Redis e processando os pagamentos, enviando requisições aos servidores externos de processamento.
- **Redis**: Banco de dados em memória utilizado como meio de comunicação entre as apis e os workers.
