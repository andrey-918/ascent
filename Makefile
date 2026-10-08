include .env

db_login:
	docker exec -it ascent-db-1 psql -U admin -d ascent

db_create_migration:
	migrate create -ext sql -dir migrations -seq $(name)

db_migrate:
	migrate -database ${DB_URL} -path migrations up