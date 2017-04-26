//#define DEBUG

/*
Title: 8x8 Switch Matrix interface
Description: Based on the last 2 switch matrix sources (for the other MIT arduino, and the Z8F.)
Debounces 64 switches one byte (8 switches) at a time using vertical counters.

Author: Jeremy Fleitz
4/10/2017 - Initial Revision


Hookup:
(Port D..0-7)
PD2 - In2
PD3 - In3
PD4 - In4
PD5 - In5
PD6 - In6
PD7 - In7

(Port B..8-13:)
PortB 
PD8 - In0
PD9 - In1

PD10 - En0
PD11 - En1
PD12 - En2

PD13 - LED out
*/

const int ledPin = 13;
const byte SW_NAK = 0x61; //a
byte strobe = 0; //0-7. This is used to output to the enable lines (driving the 74ls138)
unsigned char debounced_state[8];
String inputString = ""; 
boolean stringComplete = false; 
boolean refresh = false; //sends the status of all switches that are down;
boolean ack = false;

void setup(){
  Serial.begin(57600);
  pinMode(ledPin,OUTPUT);
  inputString.reserve(200);
  
  //setup port directions
  DDRD =  DDRD & B00000011; //Top 6 bits, PD2-PD7 are inputs
  DDRB =  DDRB & B11111100; //Bottom 2 bits are inputs
  DDRB =  DDRB | B00111100; //3 enable lines plus the led is an output
  
  strobe = 0;
  
  #ifdef DEBUG
  Serial.println("switchmatrix initialized");
  #endif
  
  ack = true;
}


void loop(){
  byte sample;
  byte curSwitches;
  byte toggle;
  
  if(ack){
    #ifdef DEBUG
      Serial.println("Ready");
    #else
      Serial.write(SW_NAK);
    #endif
    ack=false;
  }
  
  //set the strobe out
  PORTB = ((strobe << 2) | (PORTB & B00100000)) ; //keep the LED set to whatever it was
  
  //read the inputs
  sample = 0x03 & PINB; //first 2 in lines off of port B
  sample = sample | (0xfc & PIND);
  
  //debounce sample
   curSwitches = debounce(sample, strobe, &toggle);
   
   if(toggle!=0x00){
     #ifdef DEBUG
     //report the switch change
     Serial.print("Change occurred: ");
     Serial.print("toggle:");
     Serial.print(toggle);
     Serial.print(" strobe:");
     Serial.print(strobe);
     Serial.print(" sample:");
     Serial.print(sample);
     Serial.println(".");
     #endif
     
     ReportSwitchChange(strobe,curSwitches,toggle);
   }
    #ifdef DEBUG
    if(strobe==0){
     if(curSwitches&0x01 > 0){
       digitalWrite(ledPin, HIGH);
     } else {
       digitalWrite(ledPin, LOW);
     }
    }
    #endif
  
  //increment the strobe
  
  //commenting out for now so that I can test.
  strobe++;
  if (strobe > 7) {
    strobe = 0;
  }
  
  delayMicroseconds(625);
  
  #ifdef DEBUG
  if(stringComplete){
     //not caring about input string value
    Serial.println("debounce values:");
    for(int i = 0; i <= 7; i++){
      Serial.print( debounced_state[i]);
    }
    Serial.println(" done.");
    
    stringComplete = false;
  }
  #endif
  
  if(refresh){
   Refresh();
   refresh = false; 
  }
  

}

byte debounce(byte strobeSample, byte strobe, byte *toggle){
    static unsigned char clock_B[8],clock_A[8];

    unsigned char delta;

    delta = strobeSample ^ debounced_state[strobe];

    clock_B[strobe] = (clock_B[strobe] ^ clock_A[strobe]) & delta;
    clock_A[strobe] = ~clock_A[strobe] & delta;


    *toggle = delta & ~(clock_A[strobe] | clock_B[strobe]);
    debounced_state[strobe] ^= *toggle;

    return debounced_state[strobe]; //debounced_state[strobe];
}

void serialEvent() {
  while (Serial.available()) {
    // get the new byte:
    char inChar = (char)Serial.read(); 
    // add it to the inputString:
    inputString += inChar;
    // if the incoming character is a newline, set a flag
    // so the main loop can do something about it:
    
    switch(inChar){
      case '3':
        stringComplete = true;
        break;
      case '1':
        refresh = true;
        break;
      case '|':
        ack = true;
        break;
    } 
  }
}


//strobe = 1-8 (the strobe number)
//switche state for the strobe
//toggle = the value (set bit(s) ) that were changed

void ReportSwitchChange(byte strobe,  byte switches, byte toggle){
      	byte curBit = 1;
	byte i;
	byte switchID=0x00;
        unsigned char reportMsg;
        
	for(i=0;i<8; i++){
		if((toggle&curBit)>0){
                        //i is numerically the switch number for the strobe. So we can take i times the strobe to get a unique switch id
                        switchID = ((strobe-1) * 8) + i; //this will be 0 - 64
                        
			//report last bit as 1 = now high, and 0 = now low
                        #ifdef DEBUG
                          if((switches&curBit)>0){
                            Serial.print(switchID); //send now, as we may have to report another change in the same strobe
                            Serial.println(":low");
                          } else {
                            Serial.print(switchID); //send now, as we may have to report another change in the same strobe
                            Serial.println(":high");
                          }
                        #else
                          if((switches&curBit)>0){
                            reportMsg = switchID << 1;
      			  } else{
                            reportMsg = (switchID << 1) | 0x01;
  			  }
                          Serial.write(reportMsg);
                        #endif

		}

		curBit=curBit<<1;
        }

}

void Refresh(){
  byte tgl;
  #ifdef DEBUG
  Serial.println("=====Refresh Report=====");
  #endif
  for(int strobe=0;strobe<=7;strobe++){
     tgl = debounced_state[strobe];
     ReportSwitchChange(strobe,debounced_state[strobe],tgl);
  }
  #ifdef DEBUG
  Serial.println("=====End of Report=====");
  #endif
}
