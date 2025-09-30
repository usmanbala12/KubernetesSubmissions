# Use the official Google Cloud SDK image which includes a properly configured gsutil
FROM google/cloud-sdk:alpine

# Install only the required packages.
RUN apk add --no-cache postgresql-client

# Copy backup script
COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

USER 1000

ENTRYPOINT ["/usr/local/bin/backup.sh"]