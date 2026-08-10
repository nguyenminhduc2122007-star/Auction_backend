# API Reference — cURL & Postman Test Scripts

**Base URL:** `http://localhost:8081/api`

> 💡 **Postman Setup:** Tạo một **Environment** tên `Auction Dev` với variable `token` (để trống) và `admin_token` (để trống). Các test script bên dưới sẽ tự động điền sau khi login.

---

## 🔐 AUTH GROUP — `/api/auth`

---

### 1. POST /auth/register

**cURL:**

```bash
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "full_name": "Nguyen Van A",
    "password": "Password123!"
  }'
```

**Postman Test Script:**

```javascript
pm.test("Status 201 Created", () => {
  pm.response.to.have.status(201);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Response has token", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.nested.property("data.token");
  pm.environment.set("token", json.data.token);
  if (json.data.user && json.data.user.user_type === "Admin") {
    pm.environment.set("admin_token", json.data.token);
  }
});
```

---

### 2. POST /auth/login

**cURL:**

```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "Password123!"
  }'
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Token saved to environment", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.nested.property("data.token");
  pm.environment.set("token", json.data.token);
  if (json.data.user && json.data.user.user_type === "Admin") {
    pm.environment.set("admin_token", json.data.token);
    console.log("Admin token saved.");
  }
});
```

---

## 📦 ITEMS GROUP — `/api/items`

---

### 3. GET /items — List Items (Public)

**cURL:**

```bash
curl -X GET http://localhost:8081/api/items \
  -H "Content-Type: application/json"
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Response contains data items array", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.property("data");
  pm.expect(json.data).to.be.an("object");
  pm.expect(json.data).to.have.property("items");
  pm.expect(json.data.items).to.be.an("array");
});
```

---

### 4. POST /items — Create Item (Auth required)

**cURL:**

```bash
curl -X POST http://localhost:8081/api/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "title": "iPhone 16 Pro Max",
    "description": "Hàng mới 100% chưa qua sử dụng",
    "price": 1200
  }'
```

**Postman Test Script:**

```javascript
pm.test("Status 201 Created", () => {
  pm.response.to.have.status(201);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Created item has id", () => {
  const json = pm.response.json();
  pm.expect(json.data).to.have.property("id");
  // Lưu item_id để dùng ở các request sau
  pm.environment.set("item_id", json.data.id);
});
```

---

### 5. GET /items/:id — Get Item Detail

**cURL:**

```bash
curl -X GET http://localhost:8081/api/items/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>"
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Item has required fields", () => {
  const json = pm.response.json();
  pm.expect(json.data).to.include.keys("id", "title", "price");
});
```

> 💡 Trong Postman URL đặt: `{{base_url}}/items/{{item_id}}`

---

### 6. PUT /items/:id — Update Item

**cURL:**

```bash
curl -X PUT http://localhost:8081/api/items/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{
    "title": "iPhone 16 Pro Max 256GB",
    "description": "Cập nhật dung lượng 256GB",
    "price": 1350
  }'
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Updated item data returned", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.property("data");
  pm.expect(json.data.title).to.include("iPhone");
});
```

---

### 7. PUT /items/:id/status — Update Item Status (Admin only)

**cURL:**

```bash
curl -X PUT http://localhost:8081/api/items/1/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -d '{
    "status": "active"
  }'
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Status field updated", () => {
  const json = pm.response.json();
  pm.expect(json.data).to.have.property("status");
});

// Test forbidden nếu dùng user token thường
// pm.test("Status 403 Forbidden if not admin", () => {
//   pm.response.to.have.status(403);
// });
```

---

### 8. DELETE /items/:id — Delete Item

**cURL:**

```bash
curl -X DELETE http://localhost:8081/api/items/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>"
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Success message returned", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.property("message");
  pm.expect(json.message).to.include("deleted");
});
```

---

## 🛡️ ADMIN GROUP — `/api/admin` (Require Admin)

> ⚠️ Tất cả các request dưới đây phải dùng **`admin_token`** — token của tài khoản có `user_type = "Admin"`.

---

### 9. GET /admin/stats — Dashboard Stats

**cURL:**

```bash
curl -X GET http://localhost:8081/api/admin/stats \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ADMIN_TOKEN>"
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Stats has expected fields", () => {
  const json = pm.response.json();
  pm.expect(json).to.have.property("data");
  pm.expect(json.data).to.be.an("object");
});

pm.test("Forbidden if not admin", () => {
  // Uncomment khi test với user token thường
  // pm.response.to.have.status(403);
});
```

---

### 10. GET /admin/users — List Users

**cURL:**

```bash
curl -X GET http://localhost:8081/api/admin/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ADMIN_TOKEN>"
```

**Postman Test Script:**

```javascript
pm.test("Status 200 OK", () => {
  pm.response.to.have.status(200);
});

pm.test("Response body is not empty", () => {
  pm.expect(pm.response.text()).to.not.be.empty;
});

pm.test("Returns array of users", () => {
  const json = pm.response.json();
  const users = json.data;
  pm.expect(users).to.be.an("array");
  pm.expect(users.length).to.be.at.least(1);
});

pm.test("User fields are present", () => {
  const json = pm.response.json();
  const users = json.data;
  if (users.length > 0) {
    pm.expect(users[0]).to.include.keys("id", "email", "user_type");
  }
});
```

---

## 🔁 Route Summary Table

| #   | Method   | Endpoint                | Auth      | Role        |
| --- | -------- | ----------------------- | --------- | ----------- |
| 1   | `POST`   | `/api/auth/register`    | ❌ None   | —           |
| 2   | `POST`   | `/api/auth/login`       | ❌ None   | —           |
| 3   | `GET`    | `/api/items`            | Optional  | Any         |
| 4   | `POST`   | `/api/items`            | ✅ Bearer | Any         |
| 5   | `GET`    | `/api/items/:id`        | ✅ Bearer | Any         |
| 6   | `PUT`    | `/api/items/:id`        | ✅ Bearer | Owner       |
| 7   | `PUT`    | `/api/items/:id/status` | ✅ Bearer | **Admin**   |
| 8   | `DELETE` | `/api/items/:id`        | ✅ Bearer | Owner/Admin |
| 9   | `GET`    | `/api/admin/stats`      | ✅ Bearer | **Admin**   |
| 10  | `GET`    | `/api/admin/users`      | ✅ Bearer | **Admin**   |

---

## ⚡ Postman Environment Variables

| Variable      | Value                          | Mô tả               |
| ------------- | ------------------------------ | ------------------- |
| `base_url`    | `http://localhost:8081/api`    | Base URL            |
| `token`       | _(auto-set after login)_       | JWT user thường     |
| `admin_token` | _(auto-set after admin login)_ | JWT admin           |
| `item_id`     | _(auto-set after create item)_ | ID của item vừa tạo |
