FROM golang:1.21-alpine

WORKDIR /app

# Copy the source
COPY backend/main ./backend/main

# Copy public files
COPY public/ ./public/

EXPOSE 443

# Run the application
# Requires to mount the volumes separately:
#    - /app/config
#    - /app/db
CMD ["./backend/main"]