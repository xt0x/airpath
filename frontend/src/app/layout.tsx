import type { Metadata } from "next";
import "@/app/globals.css";
import { Geist } from "next/font/google";
import { cn } from "@/lib/utils";
import { TooltipProvider } from "@/components/ui/tooltip";

const geist = Geist({ subsets: ["latin"], variable: "--font-sans" });
const SHADCN_DARK_BACKGROUND = "oklch(0.145 0 0)";

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
    <html
      lang="en"
      className={cn("dark font-sans", geist.variable)}
      style={{ backgroundColor: SHADCN_DARK_BACKGROUND }}
    >
      <body style={{ backgroundColor: SHADCN_DARK_BACKGROUND }}>
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  );
}
