# All the practice code in the Learn HTTP Servers course

## Docker guide

Create a `.env` from [`.env.example`](./.env.example). Then execute the following commands to start the services

```sh
docker compose up -d
```

## DB

Connect to the db outside of Docker:

```sh
psql -h localhost -p 5432 -U postgres
```

## App

Before building the image, you need to change the value of `DB_HOST` in `.env` to `db` so the app container can call the PostgreSQL instace under Docker.
