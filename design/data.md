# Data

What the site keeps, for how long, and who can see it. This says nothing about how or where it's stored (D-002). Any storage that can keep these records and enforce these rules will do.

## Records

| Record | What it holds | Who sees it | Kept until |
|--------|---------------|-------------|------------|
| **Event** | Title, description, start date and time, optional end time, location (free text), optional capacity, draft or published, who last changed it and when | Published events: everyone. Drafts: organizers. | An organizer deletes it |
| **Sign-up** | Which event, the person's name and email address, how many people are coming (1–6), when they signed up | Organizers | 30 days after the event ends, then deleted automatically (D-004) |
| **Organizer** | Email address, active or not, who added them and when | Organizers | Removed by another organizer |
| **Sign-in link** | A hash of a one-time code, which organizer it's for, when it expires, whether it has been used | Nobody (the site only) | 1 day after it expires |
| **Session** | A hash of the session code, which organizer, when it expires | Nobody (the site only) | When it expires |
| **Activity entry** | Which organizer did what (for example *event published*, *sign-up removed*), to which event, and when. It holds no visitor names or email addresses. | Organizers | 1 year |

## Not stored
- **Contact form messages.** They're emailed to the organizers and not kept (D-005).
- **The raw codes** in sign-in links and session cookies. Only their hashes are kept, so a copy of the stored data can't be used to sign in.
- **Passwords.** There aren't any.
- **Visitors' IP addresses**, beyond whatever short-lived logs the hosting keeps for security.

## Rules
- **Times:** every date and time is in the club's time zone, a single setting. Pages never show a time zone choice (D-006).
- **Deletion:** removing an event also removes its sign-ups.
- **Export:** organizers can download the sign-ups for one event as a CSV file, for printing a check-in list.
- **Nothing personal in logs:** names, email addresses and message text never appear in application logs or activity entries.
