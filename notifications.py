# subtitlarr/notifications.py
import requests
import logging

def send_notification(webhook_url, message, title="Subtitlarr Notification", include_error=False):
    """
    Sends a notification to a webhook URL, attempting to auto-detect the type.

    Args:
        webhook_url (str): The URL of the webhook.
        message (str): The main content of the message.
        title (str, optional): The title for the notification. Defaults to "Subtitlarr Notification".
        include_error (bool, optional): If True, formats the message as an error.

    Returns:
        bool: True if the notification was sent successfully, False otherwise.
    """
    if not webhook_url:
        logging.warning("Webhook URL is not configured. Skipping notification.")
        return False

    headers = {"Content-Type": "application/json"}
    payload = {}

    # --- Auto-detect Webhook Type ---
    if "discord.com" in webhook_url:
        # Discord Webhook
        embed = {
            "title": f"**{title}**",
            "description": message,
            "color": 15158332 if include_error else 3066993  # Red for error, Green for success
        }
        payload = {"embeds": [embed]}

    elif "hooks.slack.com" in webhook_url:
        # Slack Webhook
        payload = {
            "attachments": [
                {
                    "fallback": f"{title}: {message}",
                    "color": "#ff0000" if include_error else "#36a64f", # Red for error, Green for success
                    "title": title,
                    "text": message
                }
            ]
        }
    else:
        # Generic Webhook (simple JSON)
        payload = {"title": title, "message": message}

    try:
        response = requests.post(webhook_url, json=payload, headers=headers, timeout=10)
        response.raise_for_status()
        logging.info(f"Successfully sent notification to {webhook_url}")
        return True
    except requests.exceptions.RequestException as e:
        logging.error(f"Failed to send webhook notification: {e}")
        return False