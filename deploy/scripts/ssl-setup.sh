# ./deploy/scripts/ssl-setup.sh
# 1. Установка nginx и certbot
ssh root@95.142.40.244 "sudo apt update && sudo apt install -y nginx certbot python3-certbot-nginx"

# 2. Создание директории для ACME challenge
ssh root@95.142.40.244 "sudo mkdir -p /var/www/html/.well-known/acme-challenge && sudo chown -R www-data:www-data /var/www/html"

# 3. Временная конфигурация для получения сертификата
ssh root@95.142.40.244 "sudo tee /etc/nginx/sites-available/certbot << 'EOF'
server {
    listen 80;
    server_name bot.gromovart.ru;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 404;
    }
}
EOF"

ssh root@95.142.40.244 "sudo ln -sf /etc/nginx/sites-available/certbot /etc/nginx/sites-enabled/ && sudo nginx -t && sudo systemctl reload nginx"

# 4. Получение SSL сертификата (нужен email)
EMAIL="gromovart@mail.ru"  # ЗАМЕНИ НА СВОЙ EMAIL
ssh root@95.142.40.244 "sudo certbot certonly --webroot --agree-tos --no-eff-email --email $EMAIL -w /var/www/html -d bot.gromovart.ru"

# 5. Основная конфигурация nginx
ssh root@95.142.40.244 "sudo tee /etc/nginx/sites-available/crypto-bot << 'EOF'
server {
    listen 80;
    server_name bot.gromovart.ru;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name bot.gromovart.ru;

    ssl_certificate /etc/letsencrypt/live/bot.gromovart.ru/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/bot.gromovart.ru/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;

    # Proxy к вебхуку (HTTPS к localhost:8443)
    location /webhook {
        proxy_pass https://localhost:8443/webhook;
        proxy_ssl_verify off;  # Игнорировать самоподписанный сертификат
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 90;
    }

    # Health check
    location /health {
        proxy_pass https://localhost:8443/health;
        proxy_ssl_verify off;
        proxy_set_header Host \$host;
    }
}
EOF"

# 6. Активация конфигурации
ssh root@95.142.40.244 "sudo rm -f /etc/nginx/sites-enabled/certbot && sudo ln -sf /etc/nginx/sites-available/crypto-bot /etc/nginx/sites-enabled/"

# 7. Проверка и перезапуск
ssh root@95.142.40.244 "sudo nginx -t && sudo systemctl reload nginx"