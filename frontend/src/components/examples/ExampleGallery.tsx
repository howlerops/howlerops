import { useState } from "react"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

import { InteractiveExample } from "./InteractiveExample"

const examples = {
  basics: [
    {
      id: "select-all",
      title: "SELECT All Columns",
      description: "Retrieve all columns from a table",
      initialQuery: "SELECT * FROM users;",
      sampleData: [
        { id: 1, name: "Alice", email: "alice@example.com", age: 28 },
        { id: 2, name: "Bob", email: "bob@example.com", age: 34 },
      ],
    },
    {
      id: "select-specific",
      title: "SELECT Specific Columns",
      description: "Choose only the columns you need",
      initialQuery: "SELECT name, email FROM users;",
      sampleData: [
        { name: "Alice", email: "alice@example.com" },
        { name: "Bob", email: "bob@example.com" },
      ],
    },
    {
      id: "where-clause",
      title: "WHERE Clause",
      description: "Filter results based on conditions",
      initialQuery: "SELECT * FROM users WHERE age > 25;",
      sampleData: [
        { id: 1, name: "Alice", email: "alice@example.com", age: 28 },
        { id: 2, name: "Bob", email: "bob@example.com", age: 34 },
      ],
      hint: "You can use comparison operators like >, <, >=, <=, =, !=",
    },
  ],
  joins: [
    {
      id: "inner-join",
      title: "INNER JOIN",
      description: "Combine rows from two tables based on a related column",
      initialQuery: `SELECT users.name, orders.order_id, orders.total
FROM users
INNER JOIN orders ON users.id = orders.user_id;`,
      sampleData: [
        { name: "Alice", order_id: 101, total: 150.00 },
        { name: "Bob", order_id: 102, total: 89.99 },
      ],
      hint: "INNER JOIN only returns rows that have matching values in both tables",
    },
    {
      id: "left-join",
      title: "LEFT JOIN",
      description: "Return all rows from the left table, with matching rows from the right",
      initialQuery: `SELECT users.name, orders.order_id
FROM users
LEFT JOIN orders ON users.id = orders.user_id;`,
      sampleData: [
        { name: "Alice", order_id: 101 },
        { name: "Bob", order_id: 102 },
        { name: "Charlie", order_id: null },
      ],
      hint: "LEFT JOIN returns all rows from the left table, even if there's no match",
    },
  ],
  aggregations: [
    {
      id: "count",
      title: "COUNT Function",
      description: "Count the number of rows",
      initialQuery: "SELECT COUNT(*) as total_users FROM users;",
      sampleData: [{ total_users: 150 }],
    },
    {
      id: "group-by",
      title: "GROUP BY",
      description: "Group rows that have the same values",
      initialQuery: `SELECT country, COUNT(*) as user_count
FROM users
GROUP BY country;`,
      sampleData: [
        { country: "USA", user_count: 45 },
        { country: "UK", user_count: 32 },
        { country: "Canada", user_count: 18 },
      ],
    },
    {
      id: "having",
      title: "HAVING Clause",
      description: "Filter groups based on aggregate functions",
      initialQuery: `SELECT country, COUNT(*) as user_count
FROM users
GROUP BY country
HAVING COUNT(*) > 20;`,
      sampleData: [
        { country: "USA", user_count: 45 },
        { country: "UK", user_count: 32 },
      ],
      hint: "HAVING is used with GROUP BY to filter aggregated results",
    },
  ],
  advanced: [
    {
      id: "subquery",
      title: "Subquery",
      description: "Use a query inside another query",
      initialQuery: `SELECT name, email
FROM users
WHERE id IN (
  SELECT user_id FROM orders WHERE total > 100
);`,
      sampleData: [
        { name: "Alice", email: "alice@example.com" },
        { name: "Bob", email: "bob@example.com" },
      ],
    },
    {
      id: "cte",
      title: "Common Table Expression (CTE)",
      description: "Create a temporary result set",
      initialQuery: `WITH high_value_customers AS (
  SELECT user_id, SUM(total) as total_spent
  FROM orders
  GROUP BY user_id
  HAVING SUM(total) > 500
)
SELECT users.name, high_value_customers.total_spent
FROM users
JOIN high_value_customers ON users.id = high_value_customers.user_id;`,
      sampleData: [
        { name: "Alice", total_spent: 750.00 },
        { name: "Bob", total_spent: 620.50 },
      ],
    },
  ],
}

export function ExampleGallery() {
  const [selectedCategory, setSelectedCategory] = useState("basics")

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">SQL Examples</h1>
        <p className="text-muted-foreground mt-2">
          Learn SQL with interactive examples you can run and modify
        </p>
      </div>

      <Tabs value={selectedCategory} onValueChange={setSelectedCategory}>
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="basics">Basics</TabsTrigger>
          <TabsTrigger value="joins">JOINs</TabsTrigger>
          <TabsTrigger value="aggregations">Aggregations</TabsTrigger>
          <TabsTrigger value="advanced">Advanced</TabsTrigger>
        </TabsList>

        <TabsContent value="basics" className="space-y-6 mt-6">
          {examples.basics.map((example) => (
            <InteractiveExample key={example.id} {...example} />
          ))}
        </TabsContent>

        <TabsContent value="joins" className="space-y-6 mt-6">
          {examples.joins.map((example) => (
            <InteractiveExample key={example.id} {...example} />
          ))}
        </TabsContent>

        <TabsContent value="aggregations" className="space-y-6 mt-6">
          {examples.aggregations.map((example) => (
            <InteractiveExample key={example.id} {...example} />
          ))}
        </TabsContent>

        <TabsContent value="advanced" className="space-y-6 mt-6">
          {examples.advanced.map((example) => (
            <InteractiveExample key={example.id} {...example} />
          ))}
        </TabsContent>
      </Tabs>
    </div>
  )
}
