---
title: Data Models
description: Auto-generated JavaScript/TypeScript classes from Go structs for type-safe data handling.
sidebar_position: 2
---

# Data Models

Wails auto-generates JavaScript/TypeScript classes from your Go structs. These models provide type-safe data structures that can be passed between Go and JavaScript seamlessly.

## Quick Start

Define a Go struct and export it as part of a method return type. Wails generates a corresponding TypeScript class.

```go title="models.go"
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

type UserService struct{}

func (s *UserService) GetUser() User {
    return User{
        Name:  "Alice",
        Email: "alice@example.com",
        Age:   30,
    }
}
```

```typescript title="frontend/src/App.ts"
import { UserService, User } from '../bindings/main';

const user = await UserService.GetUser();
console.log(user.name);  // "Alice"
console.log(user.email); // "alice@example.com"
```

## JSON Tags

JSON struct tags control the generated field names.

```go
type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    IsAvailable bool    `json:"isAvailable"`
}
```

Generates:

```typescript
class Product {
    id: number;
    name: string;
    price: number;
    isAvailable: boolean;
}
```

Without JSON tags, the field names are used as-is (capitalised).

## Comments and JSDoc

Go comments on struct fields become JSDoc documentation in the generated TypeScript.

```go
type Invoice struct {
    // The unique identifier for this invoice
    ID int `json:"id"`
    // Total amount in cents (e.g. 1500 = $15.00)
    Amount int `json:"amount"`
    // ISO 8601 date string
    Date string `json:"date"`
}
```

Generates:

```typescript
class Invoice {
    /** The unique identifier for this invoice */
    id: number;
    /** Total amount in cents (e.g. 1500 = $15.00) */
    amount: number;
    /** ISO 8601 date string */
    date: string;
}
```

## Nested Structs

```go
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
    Zip    string `json:"zip"`
}

type User struct {
    Name    string  `json:"name"`
    Address Address `json:"address"`
}
```

Generates:

```typescript
class Address {
    street: string;
    city: string;
    zip: string;
}

class User {
    name: string;
    address: Address;
}
```

## Arrays and Slices

```go
type Team struct {
    Name    string   `json:"name"`
    Members []string `json:"members"`
}
```

Generates:

```typescript
class Team {
    name: string;
    members: string[];
}
```

## Maps

```go
type Config struct {
    Settings map[string]string `json:"settings"`
}
```

Generates:

```typescript
class Config {
    settings: Record<string, string>;
}
```

## Type Mapping

| Go Type | TypeScript Type |
|---------|----------------|
| `string` | `string` |
| `bool` | `boolean` |
| `int`, `int8`, `int16`, `int32`, `int64` | `number` |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | `number` |
| `float32`, `float64` | `number` |
| `[]T` | `T[]` |
| `map[K]V` | `Record<K, V>` |
| `*T` | `T \| null` |
| `struct` | Generated class |
| `time.Time` | `string` |

## Creating Instances

```typescript
import { User } from '../bindings/main';

// From a plain object
const user = new User();
user.name = "Alice";
user.email = "alice@example.com";
user.age = 30;

// From JSON
const user2 = User.fromJSON({
    name: "Bob",
    email: "bob@example.com",
    age: 25,
});
```

## Passing to Go

```typescript
const user = new User();
user.name = "Charlie";
user.email = "charlie@example.com";
user.age = 35;

await UserService.UpdateUser(user);
```

## Receiving from Go

```typescript
const user = await UserService.GetUser();
// user is a fully typed User instance
console.log(user instanceof User); // true
```

## TypeScript Support

All generated models include full TypeScript type information. Import them for type annotations:

```typescript
import { User, Product, Team } from '../bindings/main';

function displayUser(user: User): void {
    console.log(`${user.name} (${user.email})`);
}

function processProducts(products: Product[]): void {
    products.forEach(p => console.log(p.name, p.price));
}
```

## Advanced Patterns

### Optional Fields

Use pointers for optional fields in Go:

```go
type Profile struct {
    Name  string  `json:"name"`
    Bio   *string `json:"bio"` // nullable
}
```

Generates:

```typescript
class Profile {
    name: string;
    bio: string | null;
}
```

### Enums with Models

```go
type Order struct {
    ID     int       `json:"id"`
    Status OrderStatus `json:"status"`
}
```

See [Enums](/docs/features/bindings/enums) for generating enum types.

### Validation

Add validation in your Go service methods:

```go
func (s *UserService) Create(user User) error {
    if user.Name == "" {
        return fmt.Errorf("name is required")
    }
    // save logic
    return nil
}
```

### Serialization

Models include `toJSON()` and `fromJSON()` methods:

```typescript
const user = await UserService.GetUser();
const json = user.toJSON();
const restored = User.fromJSON(json);
```

## See Also

- [Method Bindings](/docs/features/bindings/methods)
- [Enums](/docs/features/bindings/enums)
- [Advanced Bindings](/docs/features/bindings/advanced)
