import {chromium,expect} from "@playwright/test";
import {spawn,spawnSync} from "node:child_process";
import {fileURLToPath} from "node:url";
import path from "node:path";
import {mkdir} from "node:fs/promises";
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),".."),api=process.env.E2E_API;
if(!api?.startsWith("http://127.0.0.1:"))throw new Error("Isolated backend required");
const origin="http://127.0.0.1:3102";
const frontend=spawn(process.execPath,[path.join(root,"node_modules/next/dist/bin/next"),"dev",path.join(root,"frontend"),"--hostname","127.0.0.1","--port","3102"],{cwd:root,windowsHide:true,detached:process.platform!=="win32",env:{...process.env,BACKEND_ORIGIN:api,NEXT_TELEMETRY_DISABLED:"1"},stdio:["ignore","pipe","pipe"]});
let logs="",browser;
for(const stream of [frontend.stdout,frontend.stderr])stream.on("data",b=>logs=(logs+b.toString()).slice(-4000));
try{
 await expect.poll(async()=>{try{return (await fetch(origin+"/login")).status}catch{return 0}},{timeout:60000}).toBe(200);
 browser=await chromium.launch({headless:true});const page=await browser.newPage({viewport:{width:1500,height:1000}});
 await page.goto(origin+"/login");
 await page.getByLabel("Email",{exact:true}).fill("owner@example.invalid");await page.getByLabel("Password",{exact:true}).fill("synthetic-password-42");await page.getByRole("button",{name:"Sign in",exact:true}).click();
 await expect(page.getByRole("heading",{name:"Account & security",exact:true})).toBeVisible();
 await page.getByRole("link",{name:"Inbox",exact:true}).click();
 const conversation=page.locator(".conversation-row").filter({hasText:"6280000000000"});await conversation.click();
 await expect(page.getByText("synthetic inbound",{exact:true})).toBeVisible();
 await expect(page.locator(".presence")).toContainText("Synthetic colleague");
 await page.getByLabel("Assigned team",{exact:true}).selectOption({label:"Inbox support"});
 await expect(page.getByLabel("Assigned team",{exact:true})).toHaveValue(/.+/);
 await page.getByLabel("Assigned colleague",{exact:true}).selectOption({label:"Inbox test"});

 await expect(page.getByText("Service window: active",{exact:false})).toBeVisible();
 await page.getByRole("button",{name:"Internal note",exact:true}).click();await page.getByLabel("Internal note",{exact:true}).fill("Synthetic private note");await page.getByRole("button",{name:"Add internal note",exact:true}).click();
 await expect(page.locator(".internal-note")).toContainText("Synthetic private note");
 await page.getByRole("button",{name:"Reply",exact:true}).click();await page.getByLabel("Text reply",{exact:true}).fill("Synthetic draft survives refresh");
 await page.reload();await page.locator(".conversation-row").filter({hasText:"6280000000000"}).click();await expect(page.getByLabel("Text reply",{exact:true})).toHaveValue("Synthetic draft survives refresh");
 await expect(page.getByText("Draft restored for this conversation.",{exact:true})).toBeVisible();
 await expect(page.locator(".pricing-guard")).toContainText("zero cost service");
 await expect(page.getByRole("button",{name:"Send text reply",exact:true})).toBeEnabled();
 await mkdir(path.join(root,".local"),{recursive:true});await page.screenshot({path:path.join(root,".local/sprint3-inbox.png"),fullPage:true});
 // This browser test uses an injected fake sender and isolated synthetic assets.
 // Lose the submit response after backend acceptance, then briefly lose
 // recovery reads. The composer must stay locked and reuse the same key.
 await page.route("**/api/v1/outbound-intents",async route=>{
   if(route.request().method()!=="POST"){await route.continue();return}
   await route.fetch();
   await route.fulfill({status:502,contentType:"application/json",body:JSON.stringify({error:{code:"SYNTHETIC_LOST_RESPONSE"}})});
 });
 let recoveryFailures=0;
 await page.route("**/api/v1/outbound-intents?client_key=*",async route=>{
   if(recoveryFailures++<2){await route.fulfill({status:503,contentType:"application/json",body:JSON.stringify({error:{code:"SYNTHETIC_RECOVERY_OUTAGE"}})});return}
   await route.continue();
 });
 await page.getByRole("button",{name:"Send text reply",exact:true}).click();
 await expect(page.getByText("Send intent: submission uncertain",{exact:true})).toBeVisible();
 await expect(page.getByLabel("Text reply",{exact:true})).toBeDisabled();
 await expect(page.getByText("Send intent: accepted",{exact:true})).toBeVisible({timeout:15000});
 await page.getByRole("button",{name:"Compose a new reply",exact:true}).click();
 await page.getByLabel("Priority",{exact:true}).selectOption("HIGH");await expect(page.getByLabel("Priority",{exact:true})).toHaveValue("HIGH");
 await page.getByLabel("Text reply",{exact:true}).fill("Must be purged on logout");
 await page.locator(".conversation-row").filter({hasText:"6280000000099"}).click();
 await page.getByLabel("Text reply",{exact:true}).fill("This reply must stay blocked");
 await expect(page.locator(".pricing-guard")).toContainText("template required");
 await expect(page.getByRole("button",{name:"Send text reply",exact:true})).toBeDisabled();
 const paid=await page.evaluate(async()=>{
   const csrf=await (await fetch("/api/v1/auth/csrf")).json();
   const list=await (await fetch("/api/v1/conversations")).json();
   const c=list.items.find(c=>c.identity_value==="6280000000000");
   return await (await fetch("/api/v1/pricing/preflight",{method:"POST",headers:{"Content-Type":"application/json","X-CSRF-Token":csrf.csrf_token},body:JSON.stringify({conversation_id:c.id,assignment_revision:c.assignment_revision,text:"Synthetic marketing must be blocked",message_type:"TEMPLATE",category:"MARKETING"})})).json();
 });expect(paid.decision.code).toBe("BILLING_CURRENCY_UNVERIFIED");
 await page.getByRole("link",{name:"Pricing",exact:true}).click();await expect(page.getByRole("heading",{name:"Rate Card Registry",exact:true})).toBeVisible();await expect(page.getByText("Paid-send authority is CLOSED.",{exact:true})).toBeVisible();
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:path.join(root,".local/sprint3-pricing-mobile.png"),fullPage:true});
 await page.getByRole("button",{name:"Sign out",exact:true}).click();await expect(page).toHaveURL(origin+"/login");
 const draftCount=await page.evaluate(()=>Object.keys(sessionStorage).filter(k=>k.startsWith("waba.inbox.v1:")).length);expect(draftCount).toBe(0);
 const anonymous=await browser.newPage();await anonymous.goto(origin+"/inbox");await expect(anonymous).toHaveURL(origin+"/login");
 console.log("PASS Inbox browser E2E: scoped thread, active window, internal note, draft restore, synthetic zero-cost intent/one provider attempt with lost-response recovery, priority, pricing registry, mobile view, logout purge and anonymous rejection. No live Meta send.");
}catch(e){console.error("Frontend output:",logs);throw e}finally{if(browser)await browser.close();if(frontend.pid){if(process.platform==="win32")spawnSync("taskkill",["/PID",String(frontend.pid),"/T","/F"],{windowsHide:true,stdio:"ignore"});else{try{process.kill(-frontend.pid,"SIGTERM")}catch{}}}}
