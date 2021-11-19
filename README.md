# netbox-webhook-distributor
This is a automatic Netbox webhook distributor. It can forward a Netbox webhook event to N number of recipients.
It uses Nats jetstream to save and replay incoming Netbox events.

## config
In order to add a recipient, it needs to be added to the list of recipients in a config.yaml file. Within the netbox_webhooks, one can define which events should be distributed. A recipient needs to provide an URL which accepts JSON data via POST and return a 200 http status code.

The netbox-webhook-distributor tries to resend an event in case of http timeout or 500 http status code. (not endlessly).

The Netbox event data is not altered and distributed as is.

### example config
```yaml
recipients:
  - name: "test01"
    url: "https://any_url.com"
    netbox_webhooks:
      site:
        - "created"
        - "updated"
        - "deleted"
      device:
        - "deleted"
  - name: "test02"
    url: "http://test.com/webhook"
    netbox_webhooks:
      site:
        - "created"
        - "deleted"
      device:
        - "deleted"
```