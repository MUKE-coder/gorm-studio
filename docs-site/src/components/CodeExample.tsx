"use client";

import { CopyButton } from "./CopyButton";

const code = `package main

import (
    "github.com/gin-gonic/gin"
    "github.com/glebarez/sqlite"
    "gorm.io/gorm"
    "github.com/MUKE-coder/gorm-studio/studio"
)

type User struct {
    ID    uint   \`gorm:"primaryKey"\`
    Name  string \`gorm:"size:100"\`
    Email string \`gorm:"uniqueIndex"\`
}

func main() {
    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    db.AutoMigrate(&User{})

    r := gin.Default()
    studio.Mount(r, db, []interface{}{&User{}})
    r.Run(":8080") // Visit http://localhost:8080/studio
}`;

export function CodeExample() {
  return (
    <section className="max-w-4xl mx-auto px-6 py-20">
      <div className="text-center mb-8">
        <h2 className="text-3xl font-bold text-text-primary mb-4">
          Get started in seconds
        </h2>
        <p className="text-text-secondary">
          Add GORM Studio to any Go application with just a few lines of code.
        </p>
      </div>
      <div className="relative group rounded-xl border border-border bg-surface-secondary overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2.5 border-b border-border bg-surface-tertiary">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-danger/60" />
            <div className="w-3 h-3 rounded-full bg-warning/60" />
            <div className="w-3 h-3 rounded-full bg-success/60" />
            <span className="ml-2 text-xs text-text-muted font-mono">main.go</span>
          </div>
          <CopyButton text={code} />
        </div>
        <pre className="p-5 overflow-x-auto text-sm leading-7 font-mono text-text-secondary">
          <code>{code}</code>
        </pre>
      </div>
    </section>
  );
}
