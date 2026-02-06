run:
	docker-compose up -d --build

down:
	docker-compose down

clean:
	docker-compose down --volumes

genrsa:
	openssl genrsa -out private.pem 2048