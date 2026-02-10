package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/MUKE-coder/gorm-studio/studio"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ─── Sample Models ──────────────────────────────────────────

type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Email     string    `gorm:"size:200;uniqueIndex;not null" json:"email"`
	Role      string    `gorm:"size:50;default:user" json:"role"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Posts     []Post    `gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Profile   *Profile  `gorm:"foreignKey:UserID" json:"profile,omitempty"`
}

type Profile struct {
	ID     uint   `gorm:"primarykey" json:"id"`
	UserID uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	Bio    string `gorm:"size:500" json:"bio"`
	Avatar string `gorm:"size:300" json:"avatar"`
	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Post struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Title     string    `gorm:"size:300;not null" json:"title"`
	Body      string    `gorm:"type:text" json:"body"`
	Published bool      `gorm:"default:false" json:"published"`
	AuthorID  uint      `gorm:"not null;index" json:"author_id"`
	Author    User      `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Comments  []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
	Tags      []Tag     `gorm:"many2many:post_tags" json:"tags,omitempty"`
}

type Comment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Body      string    `gorm:"type:text;not null" json:"body"`
	PostID    uint      `gorm:"not null;index" json:"post_id"`
	AuthorID  uint      `gorm:"not null;index" json:"author_id"`
	Post      Post      `gorm:"foreignKey:PostID" json:"post,omitempty"`
	Author    User      `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Name  string `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Color string `gorm:"size:7;default:#6c5ce7" json:"color"`
	Posts []Post `gorm:"many2many:post_tags" json:"posts,omitempty"`
}

// ─── Main ───────────────────────────────────────────────────

func main() {
	// Connect to SQLite database
	db, err := gorm.Open(sqlite.Open("demo.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate
	db.AutoMigrate(&User{}, &Profile{}, &Post{}, &Comment{}, &Tag{})

	// Seed sample data
	seedData(db)

	// Create Gin router
	router := gin.Default()

	// Mount GORM Studio
	models := []interface{}{
		&User{},
		&Profile{},
		&Post{},
		&Comment{},
		&Tag{},
	}

	err = studio.Mount(router, db, models, studio.Config{
		Prefix:     "/studio",
		ReadOnly:   false,
		DisableSQL: false,
	})
	if err != nil {
		log.Fatal("Failed to mount studio:", err)
	}

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║           🗄️  GORM Studio                    ║")
	fmt.Println("║                                              ║")
	fmt.Println("║   Open: http://localhost:8080/studio          ║")
	fmt.Println("║                                              ║")
	fmt.Println("╚══════════════════════════════════════════════╝")

	router.Run(":8080")
}

// ─── Seed Data ──────────────────────────────────────────────

func seedData(db *gorm.DB) {
	var count int64
	db.Model(&User{}).Count(&count)
	if count > 0 {
		return // Already seeded
	}

	names := []string{"Alice Johnson", "Bob Smith", "Charlie Brown", "Diana Prince", "Eve Wilson", "Frank Castle", "Grace Hopper", "Henry Ford", "Iris Chen", "Jack Ryan"}
	roles := []string{"admin", "user", "editor", "moderator", "user"}
	bios := []string{"Software engineer", "Product designer", "Data scientist", "Full stack developer", "DevOps engineer", "ML researcher", "Backend developer", "Frontend developer", "CTO", "Intern"}

	// Create users
	users := make([]User, len(names))
	for i, name := range names {
		users[i] = User{
			Name:   name,
			Email:  fmt.Sprintf("%s@example.com", slugify(name)),
			Role:   roles[rand.Intn(len(roles))],
			Active: rand.Float32() > 0.2,
		}
	}
	db.Create(&users)

	// Create profiles
	for i, user := range users {
		db.Create(&Profile{
			UserID: user.ID,
			Bio:    bios[i],
			Avatar: fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", slugify(user.Name)),
		})
	}

	// Create tags
	tagNames := []string{"Go", "GORM", "Tutorial", "API", "Database", "REST", "GraphQL", "Docker", "Kubernetes", "Testing"}
	tagColors := []string{"#6c5ce7", "#00b894", "#fdcb6e", "#e17055", "#74b9ff", "#a29bfe", "#55efc4", "#fab1a0", "#81ecec", "#dfe6e9"}
	tags := make([]Tag, len(tagNames))
	for i, name := range tagNames {
		tags[i] = Tag{Name: name, Color: tagColors[i]}
	}
	db.Create(&tags)

	// Create posts
	titles := []string{
		"Getting Started with GORM",
		"Building REST APIs in Go",
		"Database Migrations Made Easy",
		"Advanced Query Techniques",
		"GORM Hooks and Callbacks",
		"Optimizing Database Performance",
		"Working with Relationships",
		"Testing GORM Applications",
		"GORM vs SQLx Comparison",
		"Building a Blog with Go and GORM",
		"Understanding GORM Scopes",
		"Soft Deletes in GORM",
		"GORM Plugin Development",
		"Connection Pooling in Go",
		"Batch Operations with GORM",
	}

	for i, title := range titles {
		post := Post{
			Title:     title,
			Body:      fmt.Sprintf("This is the content of the post about %s. It covers important concepts and best practices.", title),
			Published: rand.Float32() > 0.3,
			AuthorID:  users[rand.Intn(len(users))].ID,
		}
		db.Create(&post)

		// Add random tags
		numTags := rand.Intn(3) + 1
		for j := 0; j < numTags; j++ {
			db.Exec("INSERT OR IGNORE INTO post_tags (post_id, tag_id) VALUES (?, ?)", post.ID, tags[rand.Intn(len(tags))].ID)
		}

		// Add comments
		numComments := rand.Intn(4)
		for j := 0; j < numComments; j++ {
			db.Create(&Comment{
				Body:     fmt.Sprintf("Great post about %s! Very helpful.", title),
				PostID:   post.ID,
				AuthorID: users[rand.Intn(len(users))].ID,
			})
		}

		_ = i
	}
}

func slugify(s string) string {
	result := ""
	for _, c := range s {
		if c == ' ' {
			result += "."
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		}
	}
	return result
}
