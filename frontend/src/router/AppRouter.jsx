import { Routes, Route } from 'react-router-dom';
import {Home} from '../page/Home';
import Simulation from '../page/Simulation'



export const AppRouter = () => {
    return (
        <Routes>
            <Route path='/' element={<Home/>}/>
            <Route path='/simulacion' element={<Simulation/>}/>
        </Routes>
    )
}

export default AppRouter;