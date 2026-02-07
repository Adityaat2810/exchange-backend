-- Create additional databases for services
CREATE DATABASE auth_db;
CREATE DATABASE orders_db;

-- Grant privileges to exchange_user
GRANT ALL PRIVILEGES ON DATABASE auth_db TO exchange_user;
GRANT ALL PRIVILEGES ON DATABASE orders_db TO exchange_user;
