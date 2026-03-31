# CI/CD Checks

Gate your pipelines on catalog quality. Exit code 0 = pass, 1 = fail.

## `check scorecard`

```bash
shoehorn check scorecard my-service --min-score 70
shoehorn check scorecard my-service --min-score 80 -o json
```

## `check entity`

```bash
shoehorn check entity my-service --has-owner
shoehorn check entity my-service --has-owner --has-docs
shoehorn check entity my-service --has-owner -o json
```
