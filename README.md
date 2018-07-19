# lookout

## Dependencies with Docker Compose

The included [Docker Compose](https://docs.docker.com/compose/) file starts [bblfshd](https://github.com/bblfsh/bblfshd) and [PostgreSQL](https://www.postgresql.org/) containers.

* bblfsd listens on `localhost:9432`
* PostgreSQL listens on `localhost:5432`, with the superuser password `example`.

Clone the repository, or download `docker-compose.yml`, and run:

```bash
docker-compose up
```