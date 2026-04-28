export default function Home() {
  return (
    <main
      style={{
        display: "grid",
        minHeight: "100vh",
        placeItems: "center",
        padding: 24,
      }}
    >
      <section
        style={{
          maxWidth: 560,
          width: "100%",
        }}
      >
        <p
          style={{
            color: "#55616f",
            fontSize: 14,
            margin: "0 0 8px",
          }}
        >
          Airpath
        </p>
        <h1
          style={{
            fontSize: 28,
            fontWeight: 650,
            lineHeight: 1.2,
            margin: 0,
          }}
        >
          Flight route workspace
        </h1>
      </section>
    </main>
  );
}
