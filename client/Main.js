/** @format */

import {ToastContainer} from 'react-toastify'
import React, {useState, useEffect} from 'react' // eslint-disable-line no-unused-vars
import {BrowserRouter as Router, Route} from 'react-router-dom'

import Portal from './Portal'
import Home from './Home'
import Record from './Record'

const service = {
  name: process.env.SERVICE_NAME || 'Planet',
  url: process.env.SERVICE_URL || 'https://github.com/fiatjaf/gravity',
  provider: {
    name: process.env.SERVICE_PROVIDER_NAME || 'gravity',
    url:
      process.env.SERVICE_PROVIDER_URL || 'https://github.com/fiatjaf/gravity'
  }
}

export const GlobalContext = React.createContext({})

export default function Main() {
  let [nodeId, setNodeId] = useState(null)

  useEffect(async () => {
    if (window.ipfs) {
      let info = await window.ipfs.id()
      setNodeId(info.ID)
    }
  })

  return (
    <>
      <ToastContainer />

      <Portal to="title" clear>
        {service.name} - IPFS Gravitational Body
      </Portal>
      <Portal to="header > h1" clear>
        {service.name}
      </Portal>
      <Portal to="header aside .name" clear>
        {service.name.toLowerCase()}
      </Portal>

      <Router>
        <GlobalContext.Provider value={{nodeId}}>
          <Route exact path="/" component={Home} />
          <Route path="/:owner/:name" component={Record} />
        </GlobalContext.Provider>
      </Router>

      <Portal to="footer" clear>
        <p>
          <a href={service.provider.url}>{service.provider.name}</a>,{' '}
          {new Date().getFullYear()}
        </p>
      </Portal>
    </>
  )
}
