/*! 
 Build based on gin-vue-admin 
 Time : 1780499816000 */
import{H as F,a as w}from"./087AC4D233B64EB0vendor-form-create.Dj_l66TJ.js";import{ag as j,o as M,c as P,Q as c,R as f,$ as x,u as q,S as z,r as V}from"./087AC4D233B64EB0vendor-image-tools.BkKmqM0E.js";import"./087AC4D233B64EB0vendor-ace.B_H-8ob6.js";const D={class:"form-designer-container"},E={class:"dialog-footer"},H=Object.assign({name:"FormGenerator"},{__name:"index",setup(T){const h=V(null),d=V(!1),v=V(""),C={fieldReadonly:!1,useTemplate:!0},b=r=>r.replace(/([A-Z])/g,"-$1").toLowerCase(),R=(r,o)=>{let n=[],m=[];const u=e=>{if(e.type==="row"){const t=e.props?Object.entries(e.props).map(([a,k])=>`:${a}="${k}"`).join(" "):"";let l=e.children?e.children.map(a=>u(a)).join(`
`):"";return`
    <el-row ${t}>${l}
    </el-row>`}if(e.type==="col"){const t=e.props?Object.entries(e.props).map(([a,k])=>`:${a}="${k}"`).join(" "):"";let l=e.children?e.children.map(a=>u(a)).join(`
`):"";return`
      <el-col ${t}>${l}
      </el-col>`}if(!e.field)return"";let i=e.type;const _={input:"el-input",inputNumber:"el-input-number",select:"el-select",radio:"el-radio-group",checkbox:"el-checkbox-group",switch:"el-switch",timePicker:"el-time-picker",datePicker:"el-date-picker",slider:"el-slider",rate:"el-rate",colorPicker:"el-color-picker",cascader:"el-cascader",upload:"el-upload"}[i]||(i.startsWith("el-")?i:`el-${i}`);let g="";if(e.props)for(const[t,l]of Object.entries(e.props))l!=null&&(typeof l=="boolean"?g+=l?` ${b(t)}`:` :${b(t)}="false"`:typeof l=="string"?g+=` ${b(t)}="${l}"`:g+=` :${b(t)}='${JSON.stringify(l)}'`);let y="";e.options&&Array.isArray(e.options)&&(i==="select"?y=e.options.map(t=>`
        <el-option label="${t.label}" value="${t.value}" />`).join("")+`
      `:i==="radio"?y=e.options.map(t=>`
        <el-radio label="${t.value}">${t.label}</el-radio>`).join("")+`
      `:i==="checkbox"&&(y=e.options.map(t=>`
        <el-checkbox label="${t.value}">${t.label}</el-checkbox>`).join("")+`
      `));let N=e.value!==void 0?e.value:i==="checkbox"?[]:null;return n.push(`  ${e.field}: ${JSON.stringify(N)}`),e.$required||e.effect&&e.effect.required?m.push(`  ${e.field}: [{ required: true, message: '${e.title}不能为空', trigger: 'blur' }]`):e.validate&&m.push(`  ${e.field}: ${JSON.stringify(e.validate)}`),`
    <el-form-item label="${e.title}" prop="${e.field}">
      <${_} v-model="formData.${e.field}"${g}>${y}</${_}>
    </el-form-item>`},p=r.map(u).join(""),s=o.form||{};let $=[];return s.labelWidth&&$.push(`label-width="${s.labelWidth}"`),s.size&&$.push(`size="${s.size}"`),s.labelPosition&&$.push(`label-position="${s.labelPosition}"`),s.hideRequiredAsterisk&&$.push("hide-required-asterisk"),`<template>
  <div>
    <el-form ref="formRef" :model="formData" :rules="rules" ${$.join(" ")}>
${p}
      <el-form-item>
        <el-button type="primary" @click="submitForm">提交</el-button>
        <el-button @click="resetForm">重置</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'

const formRef = ref(null)

const formData = reactive({
${n.join(`,
`)}
})

const rules = reactive({
${m.join(`,
`)}
})

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate((valid) => {
    if (valid) {
      ElMessage.success('表单校验通过，准备提交')
      console.log('提交的数据: ', formData)
    } else {
      ElMessage.error('表单校验失败')
    }
  })
}

const resetForm = () => {
  if (!formRef.value) return
  formRef.value.resetFields()
}
<\/script>
`},S=()=>{const r=h.value.getRule(),o=h.value.getOption();v.value=R(r,o),d.value=!0},O=async()=>{try{await navigator.clipboard.writeText(v.value),w.success("代码已成功复制到剪贴板！"),d.value=!1}catch{w.error("复制失败，请手动选择复制")}};return(r,o)=>{const n=j("el-button"),m=j("el-input"),u=j("el-dialog");return M(),P("div",D,[c(q(F),{ref_key:"designer",ref:h,config:C,height:"calc(100vh - 160px)"},{handle:f(()=>[c(n,{type:"primary",size:"small",plain:"",onClick:S},{default:f(()=>[...o[3]||(o[3]=[x(" 解析为 Vue 原生标签 ",-1)])]),_:1})]),_:1},512),c(u,{modelValue:d.value,"onUpdate:modelValue":o[2]||(o[2]=p=>d.value=p),title:"生成的 Vue 模板代码",width:"70%",top:"5vh"},{footer:f(()=>[z("span",E,[c(n,{onClick:o[1]||(o[1]=p=>d.value=!1)},{default:f(()=>[...o[4]||(o[4]=[x("关闭",-1)])]),_:1}),c(n,{type:"primary",onClick:O},{default:f(()=>[...o[5]||(o[5]=[x("一键复制",-1)])]),_:1})])]),default:f(()=>[c(m,{type:"textarea",rows:25,modelValue:v.value,"onUpdate:modelValue":o[0]||(o[0]=p=>v.value=p),readonly:"",class:"code-input",resize:"none"},null,8,["modelValue"])]),_:1},8,["modelValue"])])}}});export{H as default};
