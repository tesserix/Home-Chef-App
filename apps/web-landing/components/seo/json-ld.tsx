/** Renders a schema.org object as a JSON-LD <script> (mirrors app/layout.tsx). */
export function JsonLd({ data }: { data: unknown }) {
  return (
    <script
      type="application/ld+json"
      // Server-rendered at build from trusted, internally-shaped data.
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
