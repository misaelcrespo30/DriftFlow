# Seedgen example output

Below are example seed outputs that illustrate the new rules for keys, domains, JSON arrays, and deduplication. These are illustrative samples (not sourced from the auth models) and show unique natural keys within each table.

## oidc_settings.seed.json
```json
[
  {
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "app": "web",
    "allowed_redirect_uris": [
      "https://web.elevitae.com/auth/callback",
      "https://web.mydailychoice.com/auth/callback"
    ],
    "allowed_scopes": ["openid", "email", "profile"],
    "allowed_root_domains": ["elevitae.com", "mydailychoice.com"],
    "allowed_redirect_subs": ["web", "backoffices", "login"],
    "default_client_id": "web"
  },
  {
    "tenant_id": "22222222-2222-2222-2222-222222222222",
    "app": "backoffices",
    "allowed_redirect_uris": [
      "https://web.elevitae.com/auth/callback",
      "https://web.mydailychoice.com/auth/callback"
    ],
    "allowed_scopes": ["openid", "email", "profile"],
    "allowed_root_domains": ["elevitae.com", "mydailychoice.com"],
    "allowed_redirect_subs": ["web", "backoffices", "login"],
    "default_client_id": "web"
  }
]
```

## oidc_clients.seed.json
```json
[
  {
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "app": "web",
    "client_key": "oidc-client-1",
    "redirect_uris": [
      "https://web.elevitae.com/auth/callback",
      "https://web.mydailychoice.com/auth/callback"
    ],
    "root_domains": ["elevitae.com", "mydailychoice.com"],
    "default_client_id": "web"
  },
  {
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "app": "login",
    "client_key": "oidc-client-2",
    "redirect_uris": [
      "https://web.elevitae.com/auth/callback",
      "https://web.mydailychoice.com/auth/callback"
    ],
    "root_domains": ["elevitae.com", "mydailychoice.com"],
    "default_client_id": "web"
  }
]
```

## services.seed.json
```json
[
  {
    "service_key": "branding",
    "name": "Branding",
    "domain": "elevitae.com"
  },
  {
    "service_key": "network",
    "name": "Network",
    "domain": "mydailychoice.com"
  }
]
```

## plans.seed.json
```json
[
  {
    "plan_key": "basic",
    "name": "Basic"
  },
  {
    "plan_key": "pro",
    "name": "Pro"
  }
]
```
