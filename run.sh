#!/bin/sh

/vator \
  -db-file="$DB_FILE" \
  -consumer-key="$CONSUMER_KEY" \
  -consumer-secret="$CONSUMER_SECRET" \
  -twilio-sid="$TWILIO_SID" \
  -twilio-token="$TWILIO_TOKEN" \
  -callback-domain="$FQDN" \
  ${TLS:+--tls}
