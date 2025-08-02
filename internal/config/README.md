# Configuration Package

Ce package gère la configuration de l'application kvstore avec des valeurs par défaut sensées et la possibilité de surcharger via des variables d'environnement.

## Variables d'environnement

| Variable | Type | Défaut | Description |
|----------|------|--------|-------------|
| `KVSTORE_HTTP_PORT` | int | 8080 | Port du serveur HTTP |
| `KVSTORE_HTTP_HOST` | string | localhost | Adresse d'écoute du serveur |
| `KVSTORE_LOG_LEVEL` | string | info | Niveau de log (debug, info, warn, error) |
| `KVSTORE_PERSISTENCE_TYPE` | string | memory | Type de persistance (memory, file, database) |
| `KVSTORE_PERSISTENCE_PATH` | string | ./kvstore.json | Chemin du fichier pour la persistance file |
| `KVSTORE_DATABASE_URL` | string | "" | URL de la base de données |

## Utilisation

```go
package main

import (
    "log"
    "kvstore/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Utiliser la configuration
    fmt.Printf("Server will start on %s\n", cfg.Address())
    
    if cfg.IsDebugEnabled() {
        log.Println("Debug mode enabled")
    }
}
```

## Types de persistance

* **memory**: Stockage en mémoire (données perdues au redémarrage)
* **file**: Stockage dans un fichier JSON
* **database**: Stockage en base de données (nécessite DATABASE_URL)

## Validation

La configuration est automatiquement validée lors du chargement :
* Le port doit être entre 1 et 65535
* L'hôte ne peut pas être vide
* Le niveau de log doit être valide
* Le type de persistance doit être valide
* Les paramètres requis selon le type de persistance doivent être fournis
