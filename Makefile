run:
	docker-compose up -d --build

down:
	docker-compose down

clean:
	docker-compose down --volumes