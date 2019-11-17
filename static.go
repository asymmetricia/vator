package main

import "github.com/cbroglie/mustache"

const preambleHtml = `<html>
<head>
<title>Vator! It's a motivator!</title>
</head>
<body>`

const signupTemplate = `
{{>preamble}}
Want to sign up? Great! First, you'll need a vator acount. After that, you'll need to connect to your Withings account. To
get started, pick a username and create a password.<br/>
{{>error}}
{{>toast}}
<form action="/signup" method="POST">
	<input type='text' name='username' placeholder='Username'><br/>
	<input type='password' name='password' placeholder='Password'><br/>
	<input type='password' name='confirm'  placeholder='Confirm Password'><br/>
	<input type='submit' value='Sign Up'>
</form>
<br/>
Or maybe you'd like to <a href='/login'>log in</a>, instead?
{{>postamble}}
`

const loginHtml = preambleHtml + `
    Welcome to vator! Vator will motivate you and be not at all creepy. It's a motivator.<br/>
    Perhaps you'd like to log in?<br/>
    <form action="/login" method="POST">
      <input type='text' name='username' placeholder='Username'><br/>
      <input type='password' name='password' placeholder='Password'><br/>
      <input type='submit' value='Log in'>
    </form>
    <br/>
    Or maybe you'd like to <a href='/signup'>sign up</a>, instead?
` + postambleHtml

const postambleHtml = `
  </body>
</html>
`

var partials = &mustache.StaticProvider{map[string]string{
	"preamble":  preambleHtml,
	"postamble": postambleHtml,
	"error":     `{{#error}}Something wasn't quite right: <strong>{{error}}</strong><br/>{{/error}}`,
	"toast":     `{{#toast}}Tada! <strong>{{toast}}</strong><br/>{{/toast}}`,
}}

const indexTemplate = `
{{>error}}
{{>toast}}
You're all set up! If something seems broken, click <a href='/reauth'>here</a> to re-authenticate to Withings.<br/>
&nbsp;<br/>
Phone Number: <form action="/phone" method="POST">
	<input type="text" name="phone" value="{{phone}}" placeholder="123 456 7890"/>
	<input type="submit" value="Save"/>
</form>
<form id="kgs" action="/kgs" method="POST">
	Use Kilograms: <input type="checkbox" name="kgs" {{#kgs}}checked{{/kgs}} onchange="document.getElementById('kgs').submit();"/>
</form>
Maybe you'd like to <a href='/measures'>view your recent measurements</a>?'<br/>
Or trigger a <a href='/summary'>Weekly Summary</a>?
`
