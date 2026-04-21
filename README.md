# bookstore-operator

Kubernetes operator pour gérer des ressources `BookStore`, construit avec [Kubebuilder](https://book.kubebuilder.io/).

## Description

Cet opérateur introduit la Custom Resource `BookStore` (groupe `store.mylab.local/v1alpha1`).  
Il crée et gère automatiquement un `Deployment` et un `Service` pour chaque instance `BookStore` définie dans le cluster.

**Image controller :** `ihsenalaya/bookstore-operator:v1`

---

## Installation via Helm

### Prérequis

- `kubectl` configuré sur le cluster cible
- `helm` v3+

### 1. Ajouter le repo Helm

```sh
helm repo add bookstore https://ihsenalaya.github.io/bookstore-operator
helm repo update
```

### 2. Installer l'opérateur

```sh
helm install bookstore-operator bookstore/bookstore-operator \
  --namespace bookstore-system \
  --create-namespace
```

### 3. Vérifier l'installation

```sh
kubectl get deployment -n bookstore-system
kubectl get crd bookstores.store.mylab.local
```

### Désinstaller

```sh
helm uninstall bookstore-operator -n bookstore-system
```

> **Note :** La CRD `bookstores.store.mylab.local` n'est pas supprimée automatiquement par `helm uninstall`.  
> Pour la supprimer : `kubectl delete crd bookstores.store.mylab.local`

---

## Utilisation

Créer une instance `BookStore` :

```yaml
apiVersion: store.mylab.local/v1alpha1
kind: BookStore
metadata:
  name: ma-librairie
  namespace: default
spec:
  name: central-bookstore
  replicas: 2
  image: nginx:1.25
  port: 80
```

```sh
kubectl apply -f bookstore.yaml
kubectl get bookstores -A
```

### Paramètres du Spec

| Champ      | Type   | Défaut     | Description                  |
|------------|--------|------------|------------------------------|
| `name`     | string | —          | Nom de la librairie (requis) |
| `replicas` | int32  | `1`        | Nombre de réplicas           |
| `image`    | string | `nginx:1.25` | Image du conteneur         |
| `port`     | int32  | `80`       | Port applicatif              |

---

## Configuration Helm

Valeurs personnalisables (`values.yaml`) :

```sh
# Changer le tag de l'image
helm install bookstore-operator bookstore/bookstore-operator \
  --namespace bookstore-system \
  --create-namespace \
  --set image.tag=v2 \
  --set replicaCount=2
```

| Paramètre            | Défaut                              | Description                  |
|----------------------|-------------------------------------|------------------------------|
| `image.repository`   | `ihsenalaya/bookstore-operator`     | Image du controller          |
| `image.tag`          | `v1`                                | Tag de l'image               |
| `image.pullPolicy`   | `IfNotPresent`                      | Politique de pull            |
| `replicaCount`       | `1`                                 | Réplicas du controller       |
| `leaderElect`        | `true`                              | Activer le leader election   |
| `metrics.enabled`    | `true`                              | Exposer les métriques        |
| `metrics.port`       | `8443`                              | Port métriques               |

---

## Développement

### Prérequis

- Go v1.24.6+
- Docker 17.03+
- kubectl v1.11.3+
- Kubebuilder

### Build & Deploy (Kustomize)

```sh
# Installer les CRDs
make install

# Lancer le controller localement
make run

# Build et push de l'image
make docker-build docker-push IMG=ihsenalaya/bookstore-operator:v1

# Déployer sur le cluster
make deploy IMG=ihsenalaya/bookstore-operator:v1
```

### Tests

```sh
make test        # tests unitaires
make test-e2e    # tests end-to-end (Kind)
```

---

## License

Copyright 2026. Licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
