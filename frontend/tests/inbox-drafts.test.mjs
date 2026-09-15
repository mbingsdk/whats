import {test} from "node:test";
import assert from "node:assert/strict";
import {draftKey,loadDraft,saveDraft,purgeDrafts} from "../lib/inbox-drafts.ts";
class MemoryStorage{
 data=new Map();get length(){return this.data.size}key(n){return [...this.data.keys()][n]??null}getItem(k){return this.data.get(k)??null}setItem(k,v){this.data.set(k,v)}removeItem(k){this.data.delete(k)}
}
test("draft boundaries separate session, user, organization, conversation and composer mode",()=>{
 const base=["session","user","org","conversation","reply"],keys=new Set([draftKey(...base)]);
 for(let i=0;i<base.length;i++){const values=[...base];values[i]+="different";keys.add(draftKey(...values))}
 assert.equal(keys.size,6);
});
test("restores text and idempotency metadata without retaining authority",()=>{
 const storage=new MemoryStorage(),key=draftKey("s","u","o","c","reply");
 saveDraft(storage,key,{text:"private draft",updatedAt:100,clientKey:"client-key",authorization:"never-store",token:"never-store"});
 assert.deepEqual(loadDraft(storage,key,101),{text:"private draft",updatedAt:100,clientKey:"client-key"});
 assert.doesNotMatch(storage.getItem(key),/never-store/);
});
test("stale, future and malformed drafts are rejected; logout purges all scopes only",()=>{
 const s=new MemoryStorage(),key=draftKey("s","u","o","c","note");
 for(const updatedAt of [-86400001,1000]){saveDraft(s,key,{text:"draft",updatedAt});assert.equal(loadDraft(s,key,100),null)}
 s.setItem(key,"broken");assert.equal(loadDraft(s,key),null);s.setItem("unrelated","keep");saveDraft(s,key,{text:"draft",updatedAt:100});purgeDrafts(s);assert.equal(s.length,1);assert.equal(s.getItem("unrelated"),"keep");
});
