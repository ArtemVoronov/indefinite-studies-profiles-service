# How to build and run
1. Set environment vars in the config `.env` e.g.:
```
#common settings
APP_PORT=3005
CORS='*'

#required for db service inside app
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_NAME=indefinite_studies_profiles_service_db
DATABASE_USER=indefinite_studies_profiles_service_user
DATABASE_PASSWORD=password
DATABASE_SSL_MODE=disable
DATABASE_QUERY_TIMEOUT_IN_SECONDS=30

#required for liquibase
DATABASE_URL=jdbc:postgresql://postgres:5432/indefinite_studies_profiles_service_db

```
2. Check `docker-compose.yml` is appropriate to config that you are going to use (e.g.`docker-compose config`)
3. Build images: `docker-compose build`
4. Run it: `docker-compose up`
5. Stop it: `docker-compose down`