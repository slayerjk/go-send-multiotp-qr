# go-multiotp-send-qr
Go: sync ldap users and send qr code to new users by email using MultiOTP server

<h2>Description</h2>

The idea is using MultiOTP server(https://github.com/multiOTP/multiotp) as a Credential Provider server.

In order uses to get theis QR codes it's necessary to automize some routines:
* resync ldap group to collect users
* generate qr codes(png) for all users in this group
* send qr codes to ldap users by their ldap attribute(email)

<h2>Workflow</h2>

Workflow is following:
<ol>
    <li>(Re)sync Ldap users(resyncMultiOTPUsers)</li>
    <li>Get list of New users(new user IF .png file in qrCodes dir doesn't exist). So if you deleting user, also delete his/her QR file.</li>
    <li>Generate QR(PNG) for new users(it's an indicator of old user for next running)</li>
    <li>Send mail to new users with PNG(users email domain will be the same as mailFrom domain)</li>
	<li>If send mail to new users is failed - del generated qr</li>
	<li>If app done successfully(no exit codes to the end) - sends list of 'succeeded' and 'failed' users lists(if one of their len is not 0) to admins(if -madmins flag is not "NONE").</li>
	<li>Send mail to admins if any exit code occurs(if -madmins flag is not "NONE").
</ol>

<h2>Flags</h2>

```
	logsDir := flag.String("log-dir", logsPathDefault, "set custom log dir")
	logsToKeep := flag.Int("keep-logs", 7, "set number of logs to keep after rotation")
	
    // multiotp flags
	multiOTPBinPath := flag.String("mpath", "/usr/local/bin/multiotp/multiotp.php", "full path to multiotp binary")
	qrCodesPath := flag.String("qrpath", "/etc/multiotp/qrcodes", "qr codes full path to save")
	usersPath := flag.String("upath", "/etc/multiotp/users", "MultiOTP users dir(*.db files)")
	tokenDescr := flag.String("tdescr", "TEST-SRV-OTP", "token description")

	// mail flags
	emailText := flag.String("etext", "Your OTP QR", "email text above QR code")
	mailHost := flag.String("mhost", "mail.example.com", "mail host(ip or hostname), must be valid")
	mailPort := flag.Int("mport", 25, "mail port")
	mailFrom := flag.String("mfrom", "multiotp@example.com", "mail from address, domain will be used as users' domain")
	mailSubject := flag.String("msubj", "Your QR Code", "mail subject, date and time will be added in the end")
	mailAdmins := flag.String("madmins", "NONE", "admins' emails separated by coma")
```

