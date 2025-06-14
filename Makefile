.PHONY: help infra-up infra-down infra-restart logs clean

# Default target
help:
	@echo "TikTok Demo Project Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  infra-up	   - Start infrastructure services"
	@echo "  infra-down	 - Stop infrastructure services"
	@echo "  infra-restart  - Restart infrastructure services"
	@echo "  logs		   - View service logs"
	@echo "  clean		  - Clean Docker resources"
	@echo "  dev			- Start development environment"
	@echo ""
	@echo "For Kratos service commands, please execute in the go-backend directory"

# Start infrastructure
infra-up:
	@echo "Starting infrastructure services..."
	docker-compose -f deployments/docker-compose.yml up -d
	@echo "Waiting for services to start..."
	sleep 10
	@echo "Checking service status..."
	docker-compose -f deployments/docker-compose.yml ps

# Stop infrastructure
infra-down:
	@echo "Stopping infrastructure services..."
	docker-compose -f deployments/docker-compose.yml down

# Restart infrastructure
infra-restart: infra-down infra-up

# View logs
logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# View logs for specific services
logs-mysql:
	docker-compose -f deployments/docker-compose.yml logs -f mysql-master

logs-redis:
	docker-compose -f deployments/docker-compose.yml logs -f redis-master

logs-kafka:
	docker-compose -f deployments/docker-compose.yml logs -f kafka

logs-minio:
	docker-compose -f deployments/docker-compose.yml logs -f minio

logs-consul:
	docker-compose -f deployments/docker-compose.yml logs -f consul

logs-go-backend:
	docker-compose -f deployments/docker-compose.yml logs -f go-backend

# Clean Docker resources
clean:
	@echo "Cleaning Docker resources..."
	docker-compose -f deployments/docker-compose.yml down -v
	docker system prune -f

# Start development environment
dev: infra-up
	@echo "Infrastructure started, please run the following command in the go-backend directory to start Kratos service:"
	@echo "cd go-backend && kratos run"

# Health check
health:
	@echo "Checking service health status..."
	@echo "MySQL: "
	@docker exec tiktok-mysql-master mysqladmin ping -h localhost -u root -ptiktok123 2>/dev/null && echo "✓ MySQL OK" || echo "✗ MySQL Failed"
	@echo "Redis: "
	@docker exec tiktok-redis-master redis-cli -a tiktok123 ping 2>/dev/null && echo "✓ Redis OK" || echo "✗ Redis Failed"
	@echo "MinIO: "
	@curl -s http://localhost:9000/minio/health/live >/dev/null && echo "✓ MinIO OK" || echo "✗ MinIO Failed"
	@echo "Consul: "
	@curl -s http://localhost:8500/v1/status/leader >/dev/null && echo "✓ Consul OK" || echo "✗ Consul Failed"
	@echo "Kafka: "
	@docker exec tiktok-kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1 && echo "✓ Kafka OK" || echo "✗ Kafka Failed"
	@echo "Go Backend HTTP: "
	@curl -s http://localhost:8000/health >/dev/null && echo "✓ Go Backend HTTP OK" || echo "✗ Go Backend HTTP Failed"
	@echo "Go Backend gRPC: "
	@docker exec tiktok-go-backend grpcurl -plaintext localhost:9000 list >/dev/null 2>&1 && echo "✓ Go Backend gRPC OK" || echo "✗ Go Backend gRPC Failed"
	@echo "Go Backend Container: "
	@docker ps --filter "name=tiktok-go-backend" --filter "status=running" --format "{{.Names}}" | grep -q "tiktok-go-backend" && echo "✓ Go Backend Container OK" || echo "✗ Go Backend Container Failed"

# Initialize development environment
init:
	@echo "Initializing development environment..."
	@echo "Checking if Docker is running..."
	@docker version >/dev/null 2>&1 || (echo "Docker is not running, please start Docker first" && exit 1)
	@echo "Creating necessary directories..."
	@mkdir -p configs scripts
	@echo "Initialization complete! Run 'make dev' to start the development environment"