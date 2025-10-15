import { api } from "./api.ts";

export const user = api["GET/api/user/me"].signal();
user.fetch();
