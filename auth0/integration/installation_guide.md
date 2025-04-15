By integrating Prelude's API v2, you gain access to a more robust and customizable SMS verification process, ensuring that authentication codes are delivered securely and consistently.

## Benefits of Using Prelude with Auth0

- **Improved SMS Deliverability** – Prelude offers high success rates for message delivery, reducing the risk of authentication failures.
- **Anti-Fraud Protection** – Prelude's advanced fraud detection system helps prevent abuse of your authentication system by identifying and blocking suspicious patterns.
- **Multi-Region Routing** – Prelude automatically routes messages through the optimal carrier network based on the destination country, ensuring fast delivery worldwide.
- **Multi-Channel Support** – Prelude supports SMS, WhatsApp, Viber and RCS channels for verification codes, giving your users more flexibility in how they receive authentication codes.
- **Seamless Integration** – Works effortlessly with Auth0's phone authentication, allowing you to set up in minutes.

This guide walks you through the integration process step-by-step, so you can enhance your Auth0 authentication system with Prelude's powerful API.

## Prerequisites

1. An Auth0 account and tenant. [Sign up for free](https://auth0.com/signup).
2. A Prelude account.

## Configure Prelude for Auth0

1. Enable the Auth0 integration by going to Settings > Integrations > Auth0 in your Prelude's Dashboard.
1. Create a new Secret Key for your Auth0 integration on Settings > API Keys and don't forget to save or copy it.

## Add the Auth0 Action

1. Select **Add Integration** (at the top of this page).
1. Read the necessary access requirements, and select **Continue**.
1. Configure the integration using the following fields:
   - Enter your Prelude Secret Key
1. Add the integration to your Library by selecting **Create**.
1. In the modal that appears, select the **Add to flow** link.
1. Drag the Action into the desired location in the flow.
1. Select **Apply Changes**.

## Activate custom SMS factor

To use the SMS factor, your tenant needs to have MFA enabled globally or required for specific contexts using rules. To learn how to enable the MFA feature, see:

- [Enable MFA](https://auth0.com/docs/secure/multi-factor-authentication/enable-mfa)
- [Customize MFA](https://auth0.com/docs/secure/multi-factor-authentication/customize-mfa)

Finally, configure the SMS factor to use the custom code and test the MFA flow.

**Note:** Once you complete the steps below, Auth0 will begin using this factor for MFA during login. Before activating the integration in production, make sure you have configured all components correctly, and [install and verify this Action on a test tenant](https://auth0.com/docs/get-started/auth0-overview/create-tenants/set-up-multiple-environments).

1. Go to **[Dashboard > Security > Multi-factor Auth](https://manage.auth0.com/select-tenant?path=/mfa)**, and select **Phone Message**.
1. In the modal that appears, select **Custom** for the delivery provider. When complete, select **Save** and close the modal.

   - **Note:** Message template is not supported yet

1. To begin using this factor, enable the SMS factor using the toggle switch.

## Test MFA flow

Trigger an MFA flow and verify that everything works as intended.

## Troubleshoot

If you do not receive the text message, look in the [tenant logs](https://auth0.com/docs/deploy-monitor/logs) for a failed SMS log entry. To learn which event types to search, see the [Log Event Type Code list](https://auth0.com/docs/deploy-monitor/logs/log-event-type-codes), or you can use the Filter control to find MFA errors.

**Make sure that:**

- The Action is in the Send Phone Message flow.
- The secrets match the ones you created in the steps above.
- Your [[TODO: Your service name]] account is active (not suspended).
- Your phone number is formatted using the [E.164 format](https://en.wikipedia.org/wiki/E.164).
