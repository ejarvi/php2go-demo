# Data

What the site keeps, for how long, and who can see it. This says nothing about how or where it's stored (D-002). Any storage that can keep these records and enforce these rules will do.

## Records

| Record | What it holds | Who sees it | Kept until |
|--------|---------------|-------------|------------|
| **Event** | Title, description, start date and time, optional end time, location (free text), optional capacity, draft or published, when it was last changed | Published events: everyone. Drafts: organizers. | An organizer deletes it |
| **Sign-up** | Which event, the person's name and email address, how many people are coming (1–6), when they signed up | Organizers | 30 days after the event ends, then deleted automatically (D-004) |
| **Session** | A hash of the session code, when it expires, and a fingerprint of the password it was started with | Nobody (the site only) | When it expires |
| **Activity entry** | What was done (for example *event published*, *sign-up removed*), to which event, and when. It holds no visitor names or email addresses. | Organizers | 1 year |

## Not stored
- **The organizers' password.** It's a deploy secret, not data (D-023).
- **The raw code** in session cookies. Only its hash is kept, so a copy of the stored data can't be used to sign in.
- **Visitors' IP addresses**, beyond whatever short-lived logs the hosting keeps for security.

## Rules
- **Times:** every date and time is in the club's time zone, a single setting. Pages never show a time zone choice (D-006).
- **Deletion:** removing an event also removes its sign-ups.
- **Export:** organizers can download the sign-ups for one event as a CSV file, for printing a check-in list.
- **Emailing people:** the site never sends email (D-022). An email address on a sign-up is there so organizers can reach people from their own mail if an event changes.
- **Nothing personal in logs:** names and email addresses never appear in application logs or activity entries.
