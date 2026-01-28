# Secrets Directory

This directory contains sensitive information that should NOT be visible to AI assistants.

## Files

- `jwt-secret.key` - Secret key for JWT token signing
- `encryption.key` - Key for encrypting note contents

## Security Demo

When running in the AI Sandbox environment (DevContainer or cli_claude):
- This directory is mounted as an empty tmpfs volume
- AI assistants cannot read these files
- But the API container CAN access them

This demonstrates how to safely develop with AI while protecting secrets.

## Production Use

In production:
- Use environment variables or secret management services
- Never commit secrets to git
- Rotate keys regularly
