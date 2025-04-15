const parser = require("ua-parser-js");
const Prelude = require("@prelude.so/sdk");

function categorizeDevice(data) {
  const osName = (data.os.name || "").toLowerCase();
  const deviceModel = (data.device.model || "").toLowerCase();
  const deviceType = (data.device.type || "").toLowerCase();

  if (deviceModel === "ipad") return "ipados";
  if (deviceModel === "macintosh") return "web";
  if (deviceModel === "iphone") return "ios";
  if (deviceType === "smarttv") return "tvos";
  if (osName.includes("android")) return "android";
  if (!deviceType && !data.device.model && !data.device.vendor) return "web";

  return undefined;
}

function getDeviceModel(ua) {
  if (!ua.device || (!ua.device.vendor && !ua.device.model)) {
    return undefined;
  }

  const { vendor, model } = ua.device;

  if (vendor) {
    return `${vendor}/${model || ""}`.trim();
  }

  return model || undefined;
}

exports.onExecuteSendPhoneMessage = async (event) => {
  const client = new Prelude({
    apiToken: event.secrets.PRELUDE_SECRET_KEY,
  });

  const { recipient, code } = event.message_options;
  const { ip, language } = event.request;

  const ua = parser(event.request.user_agent);

  const correlationId = `auth0:${event.tenant.id}:${event.user.user_id}`;

  await client.verification.create({
    target: {
      type: "phone_number",
      value: recipient.replace(/[\s-]/g, ""),
    },
    metadata: {
      correlation_id: correlationId,
    },
    signals: {
      ip,
      os_version: ua.os.version,
      device_model: getDeviceModel(ua),
      device_platform: categorizeDevice(ua),
    },
    options: {
      locale: language,
      custom_code: code,
    },
  });
};
