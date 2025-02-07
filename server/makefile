
migration:
	@read -p "Enter migration name: " name; \
		migrate create -ext sql -dir internal/db/sql/migrations $$name

migrate:
	migrate  -source file://internal/db/sql/migrations \
		-database postgres://postgres:1234@127.0.0.1:5432/vps_pilot?sslmode=disable up

rollback:
	migrate -source file://internal/db/sql/migrations \
		-database postgres://postgres:1234@127.0.0.1:5432/vps_pilot?sslmode=disable down

drop:
	migrate -source file://internal/db/sql/migrations \
		-database postgres://postgres:1234@127.0.0.1:5432/vps_pilot?sslmode=disable drop

sqlc:
	sqlc generate


migratef:
	@read -p "Enter migration number: " num; \
	migrate -source file://internal/db/sql/migrations \
		-database postgres://postgres:1234@127.0.0.1:5432/vps_pilot?sslmode=disable force $$num

		