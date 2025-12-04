# go-multiotp-send-qr
Go: sync ldap users and send qr code to new users by email using MultiOTP server

<h2>Description</h2>

The idea is using MultiOTP server(https://github.com/multiOTP/multiotp) as a Credential Provider server.

In order uses to get theis QR codes it's necessary to automize some routines:
* resync ldap group to collect users
* generate qr codes(png/svg) for all users in this group
* send qr codes to ldap users by their ldap attribute(email)

<h2>Workflow</h2>

Workflow is following:
<ol>
    <li>(Re)sync Ldap users(resyncMultiOTPUsers)</li>
    <li>Get list of New users with thier LDAP/AD attribute for email</li>
    <li>Generate QR(PNG) for new users</li>
    <li>Send mail to new users with PNG</li>
    <li>(Optional)Send report to admins</li>
</ol>