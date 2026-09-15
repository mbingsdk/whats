export const draftPrefix="waba.inbox.v1:";
export type Draft={text:string;updatedAt:number;clientKey?:string;intentId?:string};
export function draftKey(session:string,user:string,org:string,conversation:string,mode:"reply"|"note"){
 return draftPrefix+[session,user,org,conversation,mode].map(encodeURIComponent).join(":");
}
export function loadDraft(storage:Storage,key:string,now=Date.now()):Draft|null{
 try{const raw=storage.getItem(key);if(!raw)return null;const value=JSON.parse(raw) as Draft;
 if(typeof value.text!=="string"||value.text.length>16000||!Number.isFinite(value.updatedAt)||value.updatedAt>now||now-value.updatedAt>86400000){storage.removeItem(key);return null;}
 return {text:value.text,updatedAt:value.updatedAt,...(typeof value.clientKey==="string"?{clientKey:value.clientKey}:{}),...(typeof value.intentId==="string"?{intentId:value.intentId}:{})};
 }catch{return null;}
}
export function saveDraft(storage:Storage,key:string,draft:Draft){try{storage.setItem(key,JSON.stringify({text:draft.text,updatedAt:draft.updatedAt,clientKey:draft.clientKey,intentId:draft.intentId}));return true;}catch{return false;}}
export function purgeDrafts(storage:Storage){try{for(let i=storage.length-1;i>=0;i--){const key=storage.key(i);if(key?.startsWith(draftPrefix))storage.removeItem(key);}}catch{/* storage may be unavailable */}}
export function invalidateDrafts(){if(typeof window==="undefined")return;purgeDrafts(window.sessionStorage);try{window.localStorage.setItem("waba.identity-invalidated",String(Date.now()));}catch{/* no content is stored here */}}
