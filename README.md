# Arcane

Arcane improves the probability of getting a first-come, first-served TicketSwap ticket by instantly attempting to add a ticket to cart whenever a notification is clicked.

More often that not, when clicking a ticket notification, the ticket is already reserved, before you even had the chance of clicking the purchase button. In the rare event that you do get to see the yellow purchase button, the time it takes to click it usually destroys your chances nonetheless. Arcane snaps this purchase button out of existence. The moment you click the notification, an attempt is made to add the ticket to your cart automatically. If it succeeds, you'll know! If it doesn't succeed, you'll know too...

While the dictionary describes Arcane as mysterious and secret, I'll give you a peek behind the curtain. Arcane is a full-on man-in-the-middle attack which intercepts all HTTPS traffic between TicketSwap's app and server. Whenever a notification is clicked, the ticket identifier is grabbed from the request to make another request instead, one which attempts to reserve this ticket. All other traffic is forwarded to the authentic TicketSwap servers.

To ease the installation on iOS, Arcane implements a DNS over HTTPS server too. When combined with a self-signed root certificate and a configuration profile, Arcane can be set up in three clicks, without having to install an app.

## Mitigations

Arcane is quite different to traditional bots. Instead of continuously polling TicketSwap's servers to check for new tickets, Arcane relies on the user to click the notification. This push-based system is therefore much more difficult to detect, as it's (almost) identical to standard network traffic.

Nevertheless, client-side mitigations do exist. The most effective of which is certificate pinning, which ensures that the iOS and Android apps connect to the authentic servers. A less effective alternative is enforcing DNSSEC. While DNSSEC does not prevent man-in-the-middle attacks of the API traffic, it does make this attack more difficult to install, as it would require a user to change the proxy settings of their device.

## FAQ

### Do I still have to click the notification?

Yeah! Arcane only reduces the time it takes between you clicking the notification and adding the ticket to your cart. Usually this saves about two seconds, which gives you a fighting chance when trying to get a ticket.

### Does Arcane work with lotteries?

Nope. As you have multiple seconds to enter a raffle, reducing the time to enter isn't necessary.
