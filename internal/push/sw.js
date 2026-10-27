// Service worker for Web Push. Served at /sw.js so its scope covers the
// whole site.

self.addEventListener("push", (event) => {
    let msg = { title: "Notification", body: "" };
    if (event.data) {
        try {
            msg = event.data.json();
        } catch {
            msg.body = event.data.text();
        }
    }

    event.waitUntil(
        self.registration.showNotification(msg.title, {
            body: msg.body,
            icon: "/push/static/icon-192.png",
            data: { url: "/push-notifications" },
        })
    );
});

self.addEventListener("notificationclick", (event) => {
    event.notification.close();
    event.waitUntil(self.clients.openWindow(event.notification.data.url));
});
