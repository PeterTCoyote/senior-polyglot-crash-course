# Go Type System & Memory Model for Seniors

## 1. Zero Values are Valid States
Unlike Java or C# where uninitialized reference variables throw `NullPointerException` (or `NullReferenceException`), Go guarantees that **every allocated variable is initialized to its zero value** (`0` for numbers, `false` for booleans, `""` for strings, and `nil` for pointers/slices/maps/interfaces).
* **Senior takeaway:** Write constructors (`NewX()`) only when invariant validation is strictly necessary. Rely on zero values for clean struct defaults.

## 2. Implicit Interfaces (Satisfaction Without `implements`)
In Java/C#, a class must explicitly state `implements Greeter`. In Go, interface satisfaction is **structural and implicit**. If a type's method set includes the exact signatures, it satisfies the interface automatically.
* **Senior takeaway:** Decouple packages. Define the interface where it is *consumed*, not where the struct is *defined*.

## 3. Value vs. Pointer Receivers (and Method Sets)
* **Value receiver (`func (s SlackBot) SendNotification(...)`):** Operates on a copy. Fine for immutable / read-only types. Then both `T` and `*T` satisfy the interface.
* **Pointer receiver (`func (u *User) UpdateEmail(...)`):** Operates on the address. Required to mutate, or to avoid copying a large struct.
* **Consistency rule:** If any method of `T` uses a pointer receiver, give **all** methods of `T` pointer receivers.
* **Method-set trap:** Pointer-receiver methods live on `*T` only. `User` is not a `Notifier`; `*User` is. `[]Notifier{user}` does not compile.
* **Copy vs pointer:** `copied := user` duplicates the struct. Mutating `user` through a pointer leaves `copied` unchanged. An interface holding `*User` *does* see later mutations.

## 4. Embedding, Not Inheritance
`AuditMetadata` is embedded in `User`. Fields (and methods) are promoted, so `u.Version` works without an explicit wrapper type. There is still no subtype relationship: a `User` is not an `AuditMetadata`.
