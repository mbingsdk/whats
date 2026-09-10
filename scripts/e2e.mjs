import {chromium,expect} from "@playwright/test";
import {spawn,spawnSync} from "node:child_process";
import {fileURLToPath} from "node:url";
import path from "node:path";
import {mkdir} from "node:fs/promises";
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),"..");
const api=process.env.E2E_API;
if(!api?.startsWith("http://127.0.0.1:"))throw new Error("Isolated E2E_API required");
const origin="http://127.0.0.1:3100";
const frontend=spawn(process.execPath,[path.join(root,"node_modules/next/dist/bin/next"),"dev",path.join(root,"frontend"),"--hostname","127.0.0.1","--port","3100"],{
 cwd:root,windowsHide:true,detached:process.platform!=="win32",
 env:{...process.env,APP_ENV:"development",BACKEND_ORIGIN:api,NEXT_TELEMETRY_DISABLED:"1"},
 stdio:["ignore","pipe","pipe"]
});
let logs="";for(const stream of [frontend.stdout,frontend.stderr])stream.on("data",b=>{logs=(logs+b.toString()).slice(-3000);});
let browser;
try{
 await expect.poll(async()=>{try{return (await fetch(origin+"/login")).status;}catch{return 0;}},{timeout:60000}).toBe(200);
 browser=await chromium.launch({headless:true});
 const owner=await browser.newContext({viewport:{width:1280,height:850}});
 const page=await owner.newPage();
 page.on("dialog",d=>d.accept());
 page.on("response",async r=>{if(r.status()>=400)console.log("HTTP",r.status(),new URL(r.url()).pathname);});
 await page.goto(origin+"/login");
 await expect(page.getByRole("heading",{name:"Sign in",exact:true})).toBeVisible();
 await page.getByRole("button",{name:"Sign in",exact:true}).click();
 await expect(page.getByLabel("Email",{exact:true})).toBeFocused();
 await mkdir(path.join(root,".local"),{recursive:true});
 await page.screenshot({path:path.join(root,".local/sprint1-login.png"),fullPage:true});
 async function login(p,email,password="synthetic-password-42"){
  await p.goto(origin+"/login");
  await p.getByLabel("Email",{exact:true}).fill(email);
  await p.getByLabel("Password",{exact:true}).fill(password);
  await p.getByRole("button",{name:"Sign in",exact:true}).click();
 }
 await login(page,"owner@example.invalid");
 await expect(page.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await page.getByRole("link",{name:"Members",exact:true}).click();
 await expect(page.getByRole("heading",{name:"Members & invitations"})).toBeVisible();
 await page.getByLabel("Employee email").fill("browser@example.invalid");
 const invitation=page.getByRole("heading",{name:"Invite an employee"}).locator("..");
 await invitation.getByRole("combobox",{name:"Role",exact:true}).selectOption({label:"Agent"});
 await invitation.getByRole("button",{name:"Send invitation"}).click();
 await expect(page.getByRole("status")).toContainText("Invitation queued");
 async function mailLink(recipient,route){
  let found="";
  await expect.poll(async()=>{
   const messages=await (await fetch(api+"/__test/mail")).json();
   const message=messages.filter(m=>m.includes("To: "+recipient)&&m.includes(route+"#token=")).at(-1);
   found=message?.match(/http:\/\/127\.0\.0\.1:3100\/[^\s]+/)?.[0]??"";
   return Boolean(found);
  },{timeout:15000}).toBe(true);
  return found;
 }
 const invitationURL=await mailLink("browser@example.invalid","/accept-invitation");
 const employee=await browser.newContext({viewport:{width:1280,height:850}});
 const ep=await employee.newPage();ep.on("dialog",d=>d.accept());
 await ep.goto(invitationURL);
 await expect(ep.getByLabel("Your name")).toBeVisible();
 await expect(ep).toHaveURL(origin+"/accept-invitation");
 await ep.getByLabel("Your name").fill("Browser Employee");
 await ep.getByLabel("New password (12–256 characters)").fill("synthetic-password-42");
 await ep.getByRole("button",{name:"Accept invitation",exact:true}).click();
 await expect(ep).toHaveURL(origin+"/login");
 await login(ep,"browser@example.invalid");
 await expect(ep.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await expect(ep.getByRole("link",{name:"Members",exact:true})).toHaveCount(0);
 await ep.goto(origin+"/members");
 await expect(ep.getByRole("status")).toHaveText("You do not have permission to view this page.");
 await page.reload();
 await expect(page.getByRole("cell",{name:"Browser Employee",exact:false})).toBeVisible();
 await page.getByRole("link",{name:"Teams",exact:true}).click();
 await page.getByLabel("Team name",{exact:true}).fill("Support");
 await page.getByRole("button",{name:"Create team",exact:true}).click();
 await expect(page.getByRole("cell",{name:"Support",exact:true})).toBeVisible();
 await page.getByRole("row").filter({hasText:"Support"}).getByRole("button",{name:"Members",exact:true}).click();
 await page.getByRole("combobox",{name:"Member",exact:true}).selectOption({label:"Browser Employee"});
 await page.getByRole("button",{name:"Add member",exact:true}).click();
 await expect(page.getByRole("status")).toHaveText("Saved.");
 await page.getByRole("link",{name:"Members",exact:true}).click();
 await expect(page.getByRole("heading",{name:"Members & invitations"})).toBeVisible();
 await expect(page.getByRole("cell",{name:"Browser Employee",exact:false})).toBeVisible();
 await page.screenshot({path:path.join(root,".local/sprint1-members.png"),fullPage:true});
 await login(ep,"browser@example.invalid");
 await expect(ep.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await ep.goto(origin+"/security");
 await ep.getByRole("button",{name:"Set up authenticator"}).click();
 const secret=await ep.getByLabel("Authenticator setup secret").inputValue();
 const otp=await (await fetch(api+"/__test/totp",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({secret})})).json();
 await ep.getByLabel("Six-digit authenticator code").fill(otp.code);
 await ep.getByRole("button",{name:"Enable MFA",exact:true}).click();
 const recovery=(await ep.getByRole("textbox",{name:"Recovery codes",exact:true}).inputValue()).split("\n");
 await ep.getByRole("button",{name:"I have saved my codes"}).click();
 await ep.getByRole("button",{name:"Sign out",exact:true}).click();
 await login(ep,"browser@example.invalid");
 await ep.getByLabel("Authenticator or recovery code").fill(recovery[0]);
 await ep.getByRole("button",{name:"Verify and sign in"}).click();
 await expect(ep.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 // A second browser session is independently revoked from the first.
 const second=await browser.newContext();const sp=await second.newPage();
 await login(sp,"browser@example.invalid");
 await sp.getByLabel("Authenticator or recovery code").fill(recovery[1]);
 await sp.getByRole("button",{name:"Verify and sign in"}).click();
 await expect(sp.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await ep.getByRole("button",{name:"Revoke all other sessions"}).click();
 await expect(ep.getByRole("status")).toHaveText("Saved.");
 await sp.reload();await expect(sp).toHaveURL(origin+"/login");
 await sp.goto(origin+"/forgot-password");
 await sp.getByLabel("Email",{exact:true}).fill("browser@example.invalid");
 await sp.getByRole("button",{name:"Send email"}).click();
 const resetURL=await mailLink("browser@example.invalid","/reset-password");
 await sp.goto(resetURL);
 await sp.getByLabel("New password (12–256 characters)").fill("replacement-password-42");
 await sp.getByRole("button",{name:"Reset password",exact:true}).click();
 await expect(sp).toHaveURL(origin+"/login");
 await ep.reload();await expect(ep).toHaveURL(origin+"/login");
 await login(ep,"browser@example.invalid","replacement-password-42");
 await ep.getByLabel("Authenticator or recovery code").fill(recovery[2]);
 await ep.getByRole("button",{name:"Verify and sign in"}).click();
 await expect(ep.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await page.reload();
 await page.getByRole("row").filter({hasText:"Browser Employee"}).getByRole("button",{name:"Deactivate",exact:true}).click();
 await expect(page.getByRole("row").filter({hasText:"Browser Employee"})).toContainText("DEACTIVATED");
 await ep.reload();
 await expect(ep.getByRole("link",{name:"Members",exact:true})).toHaveCount(0);
 const session=await ep.request.get(origin+"/api/v1/auth/session");
 expect(session.status()).toBe(401);
 await expect(ep).toHaveURL(origin+"/login");
 const cookieList=await employee.cookies();
 const sessionCookie=cookieList.find(c=>c.name==="waba_session");
 expect(sessionCookie?.httpOnly).toBe(true);
 await ep.setViewportSize({width:390,height:844});
 await ep.goto(origin+"/login");
 await expect(ep.getByRole("heading",{name:"Sign in",exact:true})).toBeVisible();
 console.log("PASS browser E2E: login, local TLS invitation mail, acceptance, permission visibility, teams, TOTP/recovery login, session revoke, reset invalidation, offboarding, mobile form.");
}catch(e){
 console.error("Frontend output:",logs);
 throw e;
}finally{
 if(browser)await browser.close();
 if(frontend.pid){if(process.platform==="win32")spawnSync("taskkill",["/PID",String(frontend.pid),"/T","/F"],{windowsHide:true,stdio:"ignore"});else{try{process.kill(-frontend.pid,"SIGTERM");}catch{}}}
}
