# z258-arquitetura-microsservicos-escabilidade

Template inicial para três serviços. Enquanto a stack não for definida, cada
Dockerfile inicia NGINX 1.31.6 com uma página placeholder.

## Executar

```sh
docker compose up --build -d
```

| Serviço   | URL local             |
|-----------|-----------------------|
| service-a | http://localhost:8081 |
| service-b | http://localhost:8082 |
| service-c | http://localhost:8083 |

```sh
docker compose down
```

Substitua `index.html` e o Dockerfile de cada serviço após escolher sua stack.
Mantenha porta interna `80` ou ajuste `compose.yml` para refletir a porta real.
