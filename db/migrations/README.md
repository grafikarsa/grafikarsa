# Database Migrations

Folder ini berisi migration SQL untuk perubahan schema database Grafikarsa.

## Migration List

| No | File | Description | Date |
|----|------|-------------|------|
| 001 | `01_add_feedback_attachments.sql` | Menambahkan support attachment untuk feedback | - |
| 006 | `006_add_feed_indexes.sql` | Menambahkan indexes untuk feed algorithm | - |
| 007 | `007_add_user_contact_info.sql` | Menambahkan contact info dan privacy settings | 2026-04-11 |

## How to Apply Migration

### Windows PowerShell

```powershell
# Method 1: Using Get-Content (Recommended)
Get-Content db/migrations/007_add_user_contact_info.sql | docker exec -i grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db

# Method 2: Copy to container then execute
docker cp db/migrations/007_add_user_contact_info.sql grafikarsa-db-dev:/tmp/migration.sql
docker exec -it grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db -f /tmp/migration.sql

# Method 3: Interactive mode
docker exec -it grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db
# Then inside psql: \i /tmp/migration.sql
```

### Linux/Mac (Bash)

```bash
# Using input redirection
docker exec -i grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db < db/migrations/007_add_user_contact_info.sql

# Or using cat
cat db/migrations/007_add_user_contact_info.sql | docker exec -i grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db
```

### Using Make Command

```bash
# Import all migrations (if integrated into db.sql)
make db-import
```

## Migration 007: User Contact Information

**Added Columns:**
- `phone` (VARCHAR 20) - Nomor telepon user
- `address` (TEXT) - Alamat lengkap user
- `show_email` (BOOLEAN, default FALSE) - Privacy setting untuk email
- `show_phone` (BOOLEAN, default FALSE) - Privacy setting untuk phone
- `show_address` (BOOLEAN, default FALSE) - Privacy setting untuk address

**Indexes:**
- `idx_users_show_email` - Partial index untuk users dengan show_email = TRUE
- `idx_users_show_phone` - Partial index untuk users dengan show_phone = TRUE
- `idx_users_show_address` - Partial index untuk users dengan show_address = TRUE

**Rollback:**

**Windows PowerShell:**
```powershell
Get-Content db/migrations/007_add_user_contact_info_rollback.sql | docker exec -i grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db
```

**Linux/Mac:**
```bash
docker exec -i grafikarsa-db-dev psql -U grafikarsa_user -d grafikarsa_db < db/migrations/007_add_user_contact_info_rollback.sql
```

## Best Practices

1. **Always test migrations in development first**
2. **Create rollback scripts for every migration**
3. **Document all changes in this README**
4. **Use descriptive file names with sequential numbers**
5. **Add comments in SQL for clarity**
6. **Consider data migration if needed (not just schema)**

## Migration Naming Convention

```
<number>_<descriptive_name>.sql
```

Example:
- `007_add_user_contact_info.sql`
- `008_add_portfolio_categories.sql`

## Rollback Naming Convention

```
<number>_<descriptive_name>_rollback.sql
```

Example:
- `007_add_user_contact_info_rollback.sql`
