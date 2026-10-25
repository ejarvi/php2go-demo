# Module: contact

A way for anyone to reach the organizers without publishing an email address.

## `/contact`
- **Fields:**
  - **Name** (required).
  - **Email** (required, so organizers can reply).
  - **Message** (required, up to 2,000 characters).
- A hidden trap field catches bots, as on sign-ups (D-011).

## `POST /contact`
- **On success:** the message is emailed to the organizers' address (a configuration setting), with the visitor's email as **Reply-To**. The page then says *Thanks, we'll get back to you.*
- **Nothing is stored** (D-005).
- **If the email can't be sent,** the form is shown again, keeping the text, with *Sorry, your message couldn't be sent. Please try again later.*
- **Invalid fields** are handled as in sign-ups.
- **A trap hit** gets the normal thank-you page, and nothing is sent.

## Acceptance tests
1. A valid message sends exactly one email to the organizers' address, with Reply-To set to the visitor's email, and returns the thank-you page.
2. A message with the trap field filled in returns the thank-you page and sends nothing.
3. A message over 2,000 characters is refused, and the text is kept in the form.
4. A message containing a line break in the name or email field can't add email headers.
5. A post without the CSRF token is refused with 403.
