#!/bin/sh


down() {
    docker-compose down
}

purge() {
    docker volume rm indefinite-studies-profiles-service_database-volume
}  

build() {
    docker-compose build api
}

start() {
    docker-compose up -d
}

tail() {
    docker-compose logs -f
}

db() {
    docker exec -it indefinite-studies-profiles-service_postgres_1 psql -U indefinite_studies_profiles_service_user -d indefinite_studies_profiles_service_db_1
}

case "$1" in
  start)
    down
    purge
    build
    start
    tail
    ;;
  stop)
    down
    ;;
  tail)
    tail
    ;;
  purge)
    down
    purge
    ;;
  db)
    db
    ;;
  *)
    echo "Usage: $0 {start|stop|purge|tail|db}"
esac