import type { Metadata } from "next";
import "./globals.css";
import "../features/flights/flight-dashboard.css";

export const metadata: Metadata = {
  title: "Airpath",
  description: "Flight route display system",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
