/*! 
 Build based on gin-vue-admin 
 Time : 1780499816000 */
import{s as a}from"./087AC4D233B64EB0index.BWg6RuSE.js";const r=(o,e)=>{const t=new FormData;return t.append("provider",o),t.append("file",e),a({url:"/amazonLogisticsLibrary/uploadWorkbook",method:"post",data:t,headers:{"Content-Type":"multipart/form-data"}})},n=o=>a({url:"/amazonLogisticsLibrary/getChannelPage",method:"post",data:o}),i=o=>a({url:"/amazonLogisticsLibrary/getChannelDetail",method:"post",data:o}),g=o=>a({url:"/amazonLogisticsLibrary/getRateRowPage",method:"post",data:o}),m=o=>a({url:"/amazonLogisticsLibrary/getVersionPage",method:"post",data:o});export{g as a,n as b,m as c,i as g,r as u};
