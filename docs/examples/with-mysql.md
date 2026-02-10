# Using GORM Studio with MySQL

This example shows how to set up GORM Studio with a MySQL database.

## Prerequisites

- A running MySQL instance (5.7+ or 8.0+)
- Go 1.21+

## Installation

```bash
go get github.com/MUKE-coder/gorm-studio/studio
go get gorm.io/driver/mysql
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/MUKE-coder/gorm-studio/studio"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

type Employee struct {
    ID         uint   `gorm:"primarykey" json:"id"`
    FirstName  string `gorm:"size:100;not null" json:"first_name"`
    LastName   string `gorm:"size:100;not null" json:"last_name"`
    Email      string `gorm:"size:200;uniqueIndex" json:"email"`
    Department string `gorm:"size:100" json:"department"`
    Salary     float64 `gorm:"type:decimal(10,2)" json:"salary"`
    ManagerID  *uint  `gorm:"index" json:"manager_id"`
}

type Project struct {
    ID          uint       `gorm:"primarykey" json:"id"`
    Name        string     `gorm:"size:200;not null" json:"name"`
    Description string     `gorm:"type:text" json:"description"`
    Active      bool       `gorm:"default:true" json:"active"`
    Employees   []Employee `gorm:"many2many:project_employees" json:"employees,omitempty"`
}

func main() {
    dsn := "root:password@tcp(127.0.0.1:3306)/myapp?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to MySQL:", err)
    }

    db.AutoMigrate(&Employee{}, &Project{})

    router := gin.Default()

    models := []interface{}{&Employee{}, &Project{}}
    err = studio.Mount(router, db, models, studio.Config{
        Prefix: "/studio",
    })
    if err != nil {
        log.Fatal("Failed to mount studio:", err)
    }

    fmt.Println("GORM Studio running at http://localhost:8080/studio")
    router.Run(":8080")
}
```

## MySQL-Specific Notes

### Connection String

Use the standard MySQL DSN format. **Important:** include `parseTime=True` for proper time handling:

```go
dsn := "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
```

### Schema Introspection

GORM Studio queries MySQL's `information_schema` to discover the database structure:

- Tables from the current database (`DATABASE()`)
- Column types use MySQL native types (`int`, `varchar(100)`, `text`, `tinyint(1)`, `decimal(10,2)`, etc.)
- Primary keys detected via `COLUMN_KEY = 'PRI'`
- Foreign keys / indexes detected via `COLUMN_KEY = 'MUL'`

### SQL Editor

MySQL-specific queries work in the SQL editor:

```sql
-- MySQL-specific queries
SHOW TABLES;
DESCRIBE employees;
SHOW CREATE TABLE employees;
SELECT * FROM employees WHERE department = 'Engineering' LIMIT 10;
EXPLAIN SELECT * FROM employees WHERE manager_id IS NOT NULL;
```

### Known Considerations

- MySQL `tinyint(1)` columns are treated as boolean by GORM
- `ENUM` types are displayed as their string representation
- `parseTime=True` is required in the DSN for `time.Time` fields to work correctly
- Self-referencing foreign keys (like `manager_id` pointing to `employees.id`) are supported
- The `DESCRIBE` keyword is recognized as a read query in the SQL editor
