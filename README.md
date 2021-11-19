# netbox-webhook-distributor
This is a automatic Netbox webhook distributor. It can forward a Netbox webhook event to N number of recipients.
It uses Nats jetstream to save and replay incoming Netbox events.

## overview
The netbox-webhook-distributor tries to resend an event in case of a ```http timeout``` or ```500 http status code```. The maximum retry/backoff strategy looks as follows:
```go
    Steps:    20,
    Duration: 10 ms,
    Factor:   2.0,
    Jitter:   0.1,
```
If all retries fail, the event will be dropped, and the next event will be processed. 
The Netbox event data is not altered and distributed as is.

Each recipient will run a Nats consumer for each Netbox webhook type (device, site etc.), which allows to replay events as needed.

Events in Nats are kept for 1 hour.

## config
In order to add a recipient, it needs to be added to the list of recipients in a config.yaml file. Within the netbox_webhooks, one can define which events should be distributed. A recipient needs to provide an URL which accepts JSON data via POST and return a 200 http status code.

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