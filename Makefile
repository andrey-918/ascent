db_login:
	docker exec -it ascent-db-1 psql -U admin -d ascent

db_migration:
	migrate create -ext sql -dir migrations -seq $(name)
