import { defineConfig } from "vite";
import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";

const PORT = parseInt(process.env.PORT ?? "8080", 10);
const url = process.env.API_URL ?? "http://localhost:" + PORT;

// https://vite.dev/config/
export default defineConfig({
  plugins: [preact(), tailwindcss()],
  server: { proxy: { "/api": url } },
});
