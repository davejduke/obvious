import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "AIAUDITOR",
  description: "Autonomous AI-powered cybersecurity audit platform",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}

