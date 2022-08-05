<html>
    <head>
    <title>GNBSIM UI</title>
    </head>
    <body>
        <h2 style="text-align: center;">GNBSIM UI</h2>
        <form action="/gnbsim/v1/executeGuiProfile" method="POST">
            <label for="pType">Profile Type:</label>
            <select id="pType" name="profileType" />
            <option value="register">Register</option>
            <option value="pdusessest">PDU Session Establishment</option>
            <option value=""anrelease>AN Release</option>
            <option value="uetriggservicereq">UE Triggered Service Req</option>
            <option value="deregister">Deregister</option> </select><br>
            <label for="pName">Profile Name:</label>
            <input type="text" id="pName" name="profileName" value= "profile9"/><br>
            <label for="gnbName">GNB Name:</label>
            <input type="text" id="gnbName" name="gnbName" value="gnb1"/><br>
            <label for="sImsi">Start Imsi:</label>
            <input type="text" id="sImsi" name="startImsi" value="208930100007497"/><br>
            <label for="ueCnt">UE Count:</label>
            <input type="number" id="ueCnt" name="ueCount" value=1/><br>
            <label for="opc">OPC:</label>
            <input type="text" id="opc" name="opc" value="981d464c7c52eb6e5036234984ad0bcf"/><br>
            <label for="KEY">KEY:</label>
            <input type="text" id="key" name="key" value="5122250214c33e723a5dd523fc145fc0"/><br>
            <label for="sqn">Sequence Number:</label>
            <input type="text" id="sqn" name="sequenceNumber" value="16f3b3f70fc2"/><br>
            <label for="enf">Enable flag:</label>
            <input type="bool" id="enf" name="enable" value=true /><br>
            <label for="mcc">MCC:</label>
            <input type="text" id="mcc" name="mcc" value="208"/><br>
            <label for="mnc">MNC:</label>
            <input type="text" id="mnc" name="mnc" value="93"/><br>
            <button type="submit">EXECUTE</button>
        </form>
    </body>
</html>
