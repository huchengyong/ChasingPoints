import App from './App'

// #ifndef VUE3
import Vue from 'vue'
import './uni.promisify.adaptor'
Vue.config.productionTip = false
App.mpType = 'app'
const app = new Vue({
  ...App
})
app.$mount()
// #endif

// #ifdef VUE3
import { createSSRApp } from 'vue'
import pinia from './store'
import { issueMatchWSTicket } from './api/match.js'
import { issueUserWSTicket } from './api/user.js'
import { configureWebSocketTickets } from './utils/websocket.js'

configureWebSocketTickets({
  issueMatchWSTicket,
  issueUserWSTicket
})

export function createApp() {
  const app = createSSRApp(App)
  app.use(pinia)
  return {
    app
  }
}
// #endif
