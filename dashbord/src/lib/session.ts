import { effect } from "@preact/signals";
import { api } from "./api.ts";
import { url } from "./router.tsx";

export const user = api["GET/api/v1/me"].signal();
const refresh = api["POST/api/v1/auth/refresh"].signal();

function setTokens(
  { access_token, refresh_token, expires_in }: {
    access_token: string;
    refresh_token: string;
    expires_in: number;
  },
) {
  localStorage.setItem("access_token", access_token);
  localStorage.setItem("refresh_token", refresh_token);
  const expires_at = Date.now() + expires_in * 1000 - 5000;
  localStorage.setItem("expires_in", expires_at.toString());
}

async function ensureValidToken() {
  const token = localStorage.getItem("access_token");
  const refreshToken = localStorage.getItem("refresh_token");
  const expiresIn = localStorage.getItem("expires_in");

  if (token && expiresIn && Date.now() < parseInt(expiresIn)) {
    return token;
  }
  if (refreshToken) {
    await refresh.fetch({ refresh_token: refreshToken });
    if (refresh.data) {
      setTokens(refresh.data);
      return refresh.data.access_token;
    } else {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("expires_in");
      user.reset();
      return null;
    }
  }
  user.reset();
  return null;
}

effect(() => {
  const { path: _ } = url;
  ensureValidToken().then((token) => {
    if (token) {
      user.fetch(undefined, {
        headers: { Authorization: `Bearer ${token}` },
      });
    } else {
      user.reset();
    }
  });
});
