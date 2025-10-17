import {
  AlertCircle,
  CheckCircle,
  Code2,
  Copy,
  Info,
  MoreVertical,
} from "lucide-preact";
import wavePollLogo from "/wave_pool.svg";
import { A, navigate, url } from "../lib/router";
import { api } from "../lib/api";
import { ensureValidToken, user } from "../lib/session";
import { useState } from "preact/hooks";
import { Dialog } from "../components/Dialog";
import { Signal } from "@preact/signals";

const options = async () => {
  const token = await ensureValidToken();
  if (!token) {
    return {};
  }
  return {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  };
};
const logout = api["DELETE/api/v1/auth/logout"].signal();
const apiKeys = api["GET/api/v1/api-keys"].signal();
const webhooks = api["GET/api/v1/webhooks"].signal();
const createWebhook = api["POST/api/v1/webhooks"].signal();
const createApiKey = api["POST/api/v1/api-keys"].signal();
const deleteApiKey = api["DELETE/api/v1/api-keys/{key_id}"].signal();
apiKeys.fetch(undefined, await options());
webhooks.fetch(undefined, await options());

const secretData = new Signal<
  {
    title: string;
    description: string;
    apiKey: string;
  } | null
>(null);

const logoutHandler = async (e: Event) => {
  e.preventDefault();
  const refreshToken = localStorage.getItem("refresh_token");
  if (refreshToken) {
    await logout.fetch({ refresh_token: refreshToken });
    if (logout.data) {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("expires_in");
      user.reset();
      navigate({ params: { nav: "login" } });
    }
  }
};

const revokeApiKey = async (keyId: string) => {
  if (keyId) {
    const token = localStorage.getItem("access_token");
    if (token) {
      await deleteApiKey.fetch({ key_id: keyId }, await options());

      if (deleteApiKey.data !== undefined) {
        // Refresh the API keys list
        apiKeys.fetch(undefined, await options());
      }
    }
  }
};

const TabButton = ({ name }: { name: string }) => (
  <A
    params={{ tab: name.toLowerCase().replace(" ", "-") }}
    className={`tab tab-bordered ${
      url.params.tab === name.toLowerCase().replace(" ", "-")
        ? "tab-active text-cyan-400 border-cyan-400 font-semibold"
        : "text-gray-500 font-semibold"
    }`}
  >
    {name}
  </A>
);

function WebhooksSection() {
  return (
    <>
      <div className="flex justify-end mb-6">
        <A
          params={{ dialog: "create-webhook" }}
          className="btn btn-sm bg-cyan-400 hover:bg-cyan-500 text-white border-none normal-case"
        >
          Add new webhook
        </A>
      </div>
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="bg-white text-gray-700 font-semibold text-base">
                  Webhook URL
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base">
                  Status
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base">
                  Security Strategy
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base">
                  Event subscriptions
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base">
                  Date created
                </th>
                <th className="bg-white"></th>
              </tr>
            </thead>
            <tbody>
              {webhooks.data?.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-4 text-gray-500">
                    No webhooks found. Add one to get started.
                  </td>
                </tr>
              )}
              {webhooks.data?.map((webhook) => (
                <tr className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="text-gray-900">{webhook.url}</td>
                  <td>
                    <div className="flex items-center gap-2">
                      {webhook.status === "active"
                        ? <CheckCircle className="w-5 h-5 text-green-500" />
                        : <AlertCircle className="w-5 h-5 text-red-500" />}
                    </div>
                  </td>
                  <td className="text-gray-700">{webhook.signing_strategy}</td>
                  <td className="text-gray-700">{webhook.events.join(", ")}</td>
                  <td className="text-gray-700">{webhook.created_at}</td>
                  <td className="text-right">
                    <button className="btn btn-ghost btn-sm">
                      <MoreVertical className="w-5 h-5 text-gray-500" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

function ApiKeysSection() {
  return (
    <>
      <div className="flex justify-end mb-6">
        <A
          params={{ dialog: "create-api-key" }}
          className="btn btn-sm bg-cyan-400 hover:bg-cyan-500 text-white border-none normal-case"
        >
          Create API Key
        </A>
      </div>

      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="overflow-x-auto">
          <table className="table w-full">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="bg-white text-gray-700 font-semibold text-base">
                  key
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base">
                  APIs
                </th>
                <th className="bg-white text-gray-700 font-semibold text-base text-right">
                  Manage
                </th>
              </tr>
            </thead>
            <tbody>
              {apiKeys.data?.length === 0 && (
                <tr>
                  <td colSpan={3} className="text-center py-4 text-gray-500">
                    No API keys found. Create one to get started.
                  </td>
                </tr>
              )}
              {apiKeys.data?.map((key) => (
                <tr className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="text-gray-900">
                    <span className="font-mono">{key.prefix}●●●●</span>
                  </td>
                  <td className="text-gray-700">{key.scopes.join(", ")}</td>
                  <td className="text-right">
                    <button
                      onClick={() => {
                        revokeApiKey(key.id);
                      }}
                      className="btn btn-sm btn-outline border-cyan-400 text-cyan-400 hover:bg-cyan-50 hover:border-cyan-500 normal-case"
                    >
                      Revoke
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  );
}

export function SelectApisModal() {
  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    createApiKey.fetch({ env: "prod", scopes: ["checkout"] }, await options())
      .then(async () => {
        if (createApiKey.data) {
          apiKeys.fetch(undefined, await options());
          secretData.value = {
            title: "API Key created",
            description:
              "Keep your API key safe. You won't be able to see it again.",
            apiKey: createApiKey.data.secret_key,
          };
          createApiKey.reset();
          navigate({ params: { dialog: "secret_modal" } });
        } else if (createApiKey.error) {
          alert(`Error: ${createApiKey.error.message}`);
        }
      });
  };

  return (
    <Dialog id="create-api-key" class="modal">
      <div className="modal-box max-w-md bg-white">
        <h3 className="font-semibold text-2xl mb-6 text-gray-900">
          New API Key
        </h3>

        <p className="text-gray-700 mb-6">
          Select the APIs that this API key will have access to:
        </p>

        <form
          id="create-api-key-form"
          onSubmit={handleSubmit}
          className="space-y-4 mb-8"
        >
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              disabled
              className="checkbox checkbox-md border-2 border-gray-300 checked:border-cyan-400 [--chkbg:theme(colors.cyan.400)] [--chkfg:white]"
            />
            <span className="text-gray-900 text-base">Balance API</span>
          </label>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={true}
              className="checkbox checkbox-md border-2 border-gray-300 checked:border-cyan-400 [--chkbg:theme(colors.cyan.400)] [--chkfg:white]"
            />
            <span className="text-gray-900 text-base">Checkout API</span>
          </label>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              disabled
              className="checkbox checkbox-md border-2 border-gray-300 checked:border-cyan-400 [--chkbg:theme(colors.cyan.400)] [--chkfg:white]"
            />
            <span className="text-gray-900 text-base">Payout API</span>
          </label>
        </form>

        <div className="flex justify-end gap-3">
          <form method="dialog">
            <button
              type="submit"
              className="btn btn-ghost text-cyan-400 hover:bg-cyan-50 normal-case"
            >
              Cancel
            </button>
          </form>
          <button
            type="submit"
            form="create-api-key-form"
            className="btn bg-cyan-400 hover:bg-cyan-500 text-white border-none normal-case disabled:bg-gray-300 disabled:text-gray-500"
          >
            Create
          </button>
        </div>
      </div>
    </Dialog>
  );
}

function AddWebhookModal() {
  const [webhookUrl, setWebhookUrl] = useState("");
  const [securityStrategy, setSecurityStrategy] = useState("SIGNING_SECRET");
  const [eventSubscriptions, setEventSubscriptions] = useState({
    "b2b.payment_received": false,
    "b2b.payment_failed": false,
    "checkout.session.completed": false,
    "checkout.session.payment_failed": false,
    "merchant.payment_received": false,
  });

  const handleCheckboxChange = (event: string) => {
    setEventSubscriptions((prev) => ({
      ...prev,
      [event]: !prev[event as keyof typeof prev],
    }));
  };

  const handleSubmit = async () => {
    const selectedEvents = Object.keys(eventSubscriptions).filter(
      (event) => eventSubscriptions[event as keyof typeof eventSubscriptions],
    );

    if (!webhookUrl) {
      alert("Please enter a webhook URL.");
      return;
    }

    if (selectedEvents.length === 0) {
      alert("Please select at least one event subscription.");
      return;
    }

    createWebhook.fetch({
      url: webhookUrl,
      signing_strategy: securityStrategy,
      events: selectedEvents,
    }, await options()).then(async () => {
      if (createWebhook.data) {
        // Refresh the webhooks list
        await webhooks.fetch(undefined, await options());
        // Reset form fields
        setWebhookUrl("");
        setSecurityStrategy("SIGNING_SECRET");
        setEventSubscriptions({
          "b2b.payment_received": false,
          "b2b.payment_failed": false,
          "checkout.session.completed": false,
          "checkout.session.payment_failed": false,
          "merchant.payment_received": false,
        });
        secretData.value = {
          title: "Webhook created",
          description:
            "Keep your webhook secret safe. You won't be able to see it again.",
          apiKey: createWebhook.data.secret,
        };
        createWebhook.reset();
        navigate({ params: { dialog: "secret_modal" } });
      } else if (createWebhook.error) {
        alert(`Error: ${createWebhook.error.message}`);
      }
    });
  };

  return (
    <Dialog class="modal" id="create-webhook">
      <div className="modal-box max-w-4xl bg-white">
        <h3 className="text-2xl font-normal text-gray-900 mb-8">
          Add new webhook
        </h3>
        <div className="mb-8">
          <label className="block mb-2">
            <span className="text-cyan-400 text-sm">Webhook URL *</span>
          </label>
          <input
            type="text"
            value={webhookUrl}
            onChange={(e) =>
              setWebhookUrl((e.target as HTMLInputElement).value)}
            placeholder="https://"
            className="input input-bordered w-full border-2 border-cyan-400 focus:border-cyan-400 focus:outline-none text-base"
          />
        </div>
        <div className="mb-8">
          <div className="flex items-center gap-2 mb-4">
            <span className="text-gray-600 text-base">Security Strategy</span>
            <Info className="w-5 h-5 text-gray-400" />
          </div>
          <div className="space-y-3">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="radio"
                name="security"
                value="SIGNING_SECRET"
                checked={securityStrategy === "SIGNING_SECRET"}
                onChange={(e) =>
                  setSecurityStrategy((e.target as HTMLInputElement).value)}
                className="radio radio-lg border-2 checked:bg-black"
              />
              <span className="text-base text-gray-900">SIGNING_SECRET</span>
            </label>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="radio"
                name="security"
                value="SHARED_SECRET"
                checked={securityStrategy === "SHARED_SECRET"}
                onChange={(e) =>
                  setSecurityStrategy((e.target as HTMLInputElement).value)}
                className="radio radio-lg border-2 checked:bg-black"
              />
              <span className="text-base text-gray-900">SHARED_SECRET</span>
            </label>
          </div>
        </div>
        <div className="mb-8">
          <div className="flex items-center gap-2 mb-4">
            <span className="text-gray-600 text-base">Event subscriptions</span>
            <Info className="w-5 h-5 text-gray-400" />
          </div>
          <div className="space-y-3">
            {Object.keys(eventSubscriptions).map((event) => (
              <label
                key={event}
                className="flex items-center gap-3 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={eventSubscriptions[
                    event as keyof typeof eventSubscriptions
                  ]}
                  onChange={() => handleCheckboxChange(event)}
                  className="checkbox checkbox-lg border-2 border-gray-300 [--chkbg:black] [--chkfg:white]"
                />
                <span className="text-base text-gray-900">{event}</span>
              </label>
            ))}
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-8">
          <form method="dialog">
            <button
              type="submit"
              className="btn btn-ghost text-gray-700 text-base px-6"
            >
              Cancel
            </button>
          </form>
          <button
            onClick={handleSubmit}
            className="btn bg-cyan-400 hover:bg-cyan-500 text-white border-none text-base px-6"
          >
            Submit
          </button>
        </div>
      </div>
    </Dialog>
  );
}

function SecretModal() {
  if (!secretData.value) return null;

  const [isSaved, setIsSaved] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(secretData.value?.apiKey || "");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleClose = () => {
    secretData.value = null;
    setIsSaved(false);
  };

  return (
    <Dialog onClose={handleClose} id="secret_modal" className="modal">
      <div className="modal-box max-w-3xl bg-white">
        <h3 className="text-2xl font-normal text-gray-900 mb-6">
          {secretData.value?.title}
        </h3>

        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
          <div className="flex gap-3">
            <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
            <div>
              <h4 className="font-semibold text-gray-900 mb-1">
                Keep your secret safe
              </h4>
              <p className="text-sm text-gray-700">
                Save and store this new secret to a secure place, such as a
                password manager or secret store. You won't be able to see it
                again.
              </p>
            </div>
          </div>
        </div>

        <div className="flex gap-2 mb-6">
          <input
            type="text"
            value={secretData.value?.apiKey || ""}
            readOnly
            className="input input-bordered flex-1 font-mono text-sm bg-gray-50 text-gray-900"
          />
          <button
            onClick={handleCopy}
            className="btn bg-white border-cyan-400 text-cyan-400 hover:bg-cyan-50"
          >
            <Copy className="w-4 h-4 mr-2" />
            {copied ? "Copied!" : "Copy secret"}
          </button>
        </div>

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={isSaved}
            onChange={(e) => setIsSaved((e.target as HTMLInputElement).checked)}
            className="checkbox checkbox-sm border-2 border-gray-300 [--chkbg:black] [--chkfg:white]"
          />
          <span
            className={`text-sm ${isSaved ? "text-gray-900" : "text-gray-400"}`}
          >
            Yes, I saved this secret
          </span>
        </label>
        <div className="modal-action">
          <button
            onClick={handleClose}
            disabled={!isSaved}
            className={`btn ${
              isSaved
                ? "bg-cyan-400 hover:bg-cyan-500 text-white"
                : "bg-gray-200 text-gray-400 cursor-not-allowed"
            } border-none`}
          >
            Done
          </button>
        </div>
      </div>
    </Dialog>
  );
}

export function DeveloperPortal() {
  const { nav, tab } = url.params;
  if (nav !== "dev-portal" || !tab) {
    navigate({ params: { nav: "dev-portal", tab: "api-keys" } });
  }
  return (
    <div className="flex min-h-screen bg-gray-50">
      <aside className="w-60 bg-white border-r border-gray-200">
        <div className="flex items-center gap-2 p-6 border-b border-gray-200">
          <div className="w-8 h-8 bg-gradient-to-br rounded-lg flex items-center justify-center">
            <img className="w-8 h-8" src={wavePollLogo} alt="Wave Pool Logo" />
          </div>
          <span className="text-xl text-gray-900 font-bold">Wave Pool</span>
        </div>

        <nav className="p-4">
          <ul className="space-y-1">
            <li>
              <A
                params={{ nav: "dev-portal" }}
                className={`flex items-center gap-3 px-4 py-3 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors ${
                  url.params.nav === "dev-portal"
                    ? "bg-gray-100 font-semibold"
                    : ""
                }`}
              >
                <Code2 className="w-5 h-5 text-blue-500" />
                <span className="font-medium">Developer Portal</span>
              </A>
            </li>
          </ul>
        </nav>
      </aside>

      <div className="flex-1 flex flex-col">
        <header className="bg-white border-b border-gray-200 px-8 py-4">
          <div className="flex items-center justify-end">
            <div
              role="button"
              onClick={logoutHandler}
              className="flex items-center gap-3 cursor-pointer hover:bg-gray-100 px-3 py-1 rounded-lg transition-colors"
            >
              <div className="text-right">
                <div className="font-semibold text-gray-900">
                  {user.data?.business.name}
                </div>
                <div className="text-sm text-gray-500">
                  {user.data?.business.country} {user.data?.business.name}
                </div>
              </div>
              <div className="avatar placeholder">
                <div className="bg-cyan-400 text-white rounded-full w-10 h-10 flex items-center justify-center">
                  <span className="text-lg font-semibold">
                    {user.data?.business.name[0]}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </header>

        <main className="flex-1 p-8">
          <div className="max-w-7xl mx-auto">
            <h1 className="text-4xl font-bold text-gray-900 mb-8">
              Developer Portal
            </h1>

            <div className="tabs tabs-bordered mb-8">
              <TabButton name="API KEYS" />
              <TabButton name="WEBHOOKS" />
            </div>

            {tab === "api-keys" && <ApiKeysSection />}
            {tab === "webhooks" && <WebhooksSection />}
          </div>
        </main>
        <AddWebhookModal />
        <SecretModal />
        <SelectApisModal />
      </div>
    </div>
  );
}
